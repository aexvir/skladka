import tokenTypes from "./token-types.js";
//#region lib/theme.ts
const defaultCssVariablePrefix = "--hls-";
/**
* Bytes of one packed Wasm theme table: 73 five-byte records padded for the
* emitter's SIMD comparisons. Mirrors `$mem.themeTable` in src/memory.wat.
*/
const themeTableBytes = 384;
const enc = new TextEncoder();
const colorReg = /^#([a-f0-9]{3,4}|[a-f0-9]{6}|[a-f0-9]{8})$/i;
const displayP3Reg = /^color\(display-p3\s+[+-]?(?:\d+(?:\.\d*)?|\.\d+)\s+[+-]?(?:\d+(?:\.\d*)?|\.\d+)\s+[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:\s*\/\s*[+-]?(?:\d+(?:\.\d*)?|\.\d+))?\s*\)$/i;
/**
* Accept hex and numeric Display P3 colors that cannot escape a CSS declaration.
* Named colors, variables, and arbitrary CSS fragments are rejected.
*/
function isThemeColor(value) {
	return typeof value === "string" && (colorReg.test(value.trim()) || displayP3Reg.test(value.trim()));
}
/** A theme's default text color: `editor.foreground`, `text`, then `foreground`. */
function themeForeground(style) {
	return style["editor.foreground"] ?? style.text ?? style.foreground;
}
/** A theme's editor background: `editor.background`, then `background`. */
function themeBackground(style) {
	return style["editor.background"] ?? style.background;
}
/** Describe a rejected theme value for the `invalid theme` error message. */
function describeThemeValue(value) {
	if (value === null) return "null";
	if (value === void 0) return "undefined";
	if (typeof value === "string") return `the string ${JSON.stringify(value)} (theme ids are not supported; import the theme object from "@pierre/highlights/themes")`;
	if (Array.isArray(value)) return "an array";
	if (typeof value === "object") {
		if ("themes" in value && Array.isArray(value.themes)) return `a ThemeFamily whose first member is ${describeThemeValue(value.themes[0])}`;
		return "an object without a `name`";
	}
	return `a ${typeof value}`;
}
/** True for an object with a non-empty `name`, the shape every theme needs. */
function isThemeObject(value) {
	return value != null && typeof value === "object" && "name" in value && typeof value.name === "string" && value.name !== "";
}
/**
* Resolve a `theme` option. For a `ThemeFamily`, use its first member.
* Throw a `TypeError` that describes the received value if it is invalid.
* Theme ID strings are not supported.
*/
function resolveTheme(theme) {
	const resolved = theme != null && typeof theme === "object" && "themes" in theme && theme.themes != null ? theme.themes[0] : theme;
	if (!isThemeObject(resolved)) throw new TypeError(`invalid theme: expected a Theme or ThemeFamily object, received ${describeThemeValue(theme)}`);
	return resolved;
}
/** Keep the nearest scope's font settings while finding an inherited color. */
function resolveThemeSyntax(syntax, name) {
	let settings;
	for (let k = name; k !== "";) {
		const v = syntax[k];
		if (typeof v === "string") return {
			...settings,
			color: v
		};
		if (v != null && typeof v === "object") {
			settings ??= v;
			if (v.color != null) return {
				...settings,
				color: v.color
			};
		}
		const dot = k.lastIndexOf(".");
		k = dot < 0 ? "" : k.slice(0, dot);
	}
	return settings;
}
/** Resolve identical colors and font settings for Wasm and JavaScript output. */
function resolveThemeStyle(style, name) {
	let color;
	let font_style;
	let font_weight;
	if (name === "background") color = themeBackground(style);
	else if (name === "foreground") color = themeForeground(style);
	else {
		({color, font_style, font_weight} = resolveThemeSyntax(style.syntax ?? {}, name) ?? {});
		if (font_style != null || font_weight != null) color ??= themeForeground(style);
	}
	if (isThemeColor(color)) {
		color = color.trim().toLowerCase();
		if (color.startsWith("#")) {
			if (color.length <= 5) color = color.replace(/[^#]/g, "$&$&");
			if (color.length === 9 && color.endsWith("ff")) color = color.slice(0, 7);
			if (color === "#00000000") color = void 0;
		}
	} else color = void 0;
	return {
		color,
		italic: font_style === "italic",
		weight: font_weight != null && font_weight >= 100 && font_weight <= 900 ? Math.round(font_weight / 100) * 100 : 0
	};
}
/** A prefixed CSS variable for a token slot, matching the Wasm emitter. */
function cssVariable(name, cssVariablePrefix) {
	return `var(${cssVariablePrefix}${name.replace(/[._]/g, "-")})`;
}
/**
* Escape a string for a double-quoted HTML attribute.
* Prefixes and theme keys need escaping. Validated theme colors do not.
*/
function escapeAttribute(value) {
	return value.replaceAll("&", "&amp;").replaceAll("\"", "&quot;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
}
/**
* Pack HTML openers for Wasm mode 4. Store the u32 span reserve, then
* u32 byte offset/length pairs per token ID, then the UTF-8 bytes.
* Slot 0 holds the root opener with `<code>`. Other slots hold span openers.
*/
function packHtmlOpeners(openers) {
	const bytes = openers.map((opener) => enc.encode(opener));
	let offset = 4 + 8 * bytes.length;
	const blob = new Uint8Array(offset + bytes.reduce((sum, entry) => sum + entry.length, 0));
	const dv = new DataView(blob.buffer);
	let spanReserve = 32;
	for (let i = 0; i < bytes.length; i++) {
		dv.setUint32(4 + 8 * i, offset, true);
		dv.setUint32(8 + 8 * i, bytes[i].length, true);
		blob.set(bytes[i], offset);
		offset += bytes[i].length;
		if (i !== 0) spanReserve = Math.max(spanReserve, bytes[i].length + 32);
	}
	dv.setUint32(0, spanReserve, true);
	return blob;
}
const preparedCache = /* @__PURE__ */ new WeakMap();
/**
* Resolve all token styles for a theme or family.
* Cache by theme object identity and, for CSS-variable themes, by prefix.
* HTML, token, stream, and live output share the prepared object.
*/
function prepareTheme(theme, cssVariablePrefix = defaultCssVariablePrefix) {
	const resolved = resolveTheme(theme);
	const prefix = resolved.cssVariables === true ? cssVariablePrefix : "";
	let prefixes = preparedCache.get(resolved);
	const cached = prefixes?.get(prefix);
	if (cached !== void 0) return cached;
	const prepared = resolved.cssVariables === true ? prepareCssVariables(resolved.name, cssVariablePrefix) : typeof resolved.cssVariables === "object" ? prepareCssPalette(resolved) : prepareStyles(resolved);
	if (prefixes === void 0) {
		prefixes = /* @__PURE__ */ new Map();
		preparedCache.set(resolved, prefixes);
	}
	prefixes.set(prefix, prepared);
	return prepared;
}
/** Palette colors are variable suffixes, not literal CSS values. */
function prepareCssPalette(theme) {
	const config = theme.cssVariables;
	if (typeof config !== "object") throw new Error("Expected CSS palette");
	const prefix = config.prefix ?? "--hls-";
	const variable = (name) => {
		const fallback = config.defaults?.[name];
		return `var(${prefix}${name}${fallback ? `, ${fallback}` : ""})`;
	};
	const styles = new Array(tokenTypes.length).fill(null);
	const syntax = theme.style.syntax ?? {};
	for (let i = 1; i < tokenTypes.length; i++) {
		const setting = resolveThemeSyntax(syntax, tokenTypes[i]);
		styles[i] = {
			color: variable(setting?.color ?? "foreground"),
			italic: setting?.font_style === "italic",
			weight: setting?.font_weight === void 0 ? 0 : Math.round(setting.font_weight / 100) * 100
		};
	}
	const fg = variable(themeForeground(theme.style) ?? "foreground");
	const bg = variable(themeBackground(theme.style) ?? "background");
	let htmlOpeners;
	return {
		name: theme.name,
		styles,
		fg,
		bg,
		table: void 0,
		get htmlOpeners() {
			return htmlOpeners ??= themeHtmlOpeners(styles, fg, bg);
		}
	};
}
/**
* Each slot uses its prefixed CSS custom property. The theme's `style`
* is not read. Build these references on first use. HTML output only
* needs to know that there is no packed theme table.
*/
function prepareCssVariables(name, cssVariablePrefix) {
	let variables;
	const resolve = () => variables ??= cssVariableStyles(cssVariablePrefix);
	return {
		name,
		get styles() {
			return resolve().styles;
		},
		get fg() {
			return resolve().fg;
		},
		get bg() {
			return resolve().bg;
		},
		table: void 0,
		htmlOpeners: void 0
	};
}
/** Variable references for every token slot plus the foreground and background. */
function cssVariableStyles(cssVariablePrefix) {
	const styles = new Array(tokenTypes.length).fill(null);
	for (let i = 1; i < tokenTypes.length; i++) styles[i] = {
		color: cssVariable(tokenTypes[i], cssVariablePrefix),
		italic: false,
		weight: 0
	};
	return {
		styles,
		fg: cssVariable("foreground", cssVariablePrefix),
		bg: cssVariable("background", cssVariablePrefix)
	};
}
/**
* Resolve token styles and pack hex colors and font settings into a Wasm
* theme table. Keep font settings even when the color is absent.
* Display P3 colors do not fit RGBA records. Themes with these colors
* use prepared HTML openers instead of a packed table.
*/
function prepareStyles(theme) {
	const themeStyle = theme.style ?? {};
	const styles = new Array(tokenTypes.length).fill(null);
	const table = new Uint8Array(tokenTypes.length * 5);
	let fg;
	let bg;
	let usesDisplayP3 = false;
	for (let i = 1; i < tokenTypes.length; i++) {
		const name = tokenTypes[i];
		const style = resolveThemeStyle(themeStyle, name);
		const { color, italic, weight } = style;
		const o = i * 5;
		if (color !== void 0) {
			if (color.startsWith("#")) {
				const rgb = parseInt(color.slice(1, 7), 16);
				table[o] = rgb >> 16;
				table[o + 1] = rgb >> 8 & 255;
				table[o + 2] = rgb & 255;
				table[o + 3] = color.length === 9 ? parseInt(color.slice(7), 16) : 255;
			} else usesDisplayP3 = true;
		}
		table[o + 4] = (italic ? 16 : 0) | weight / 100;
		if (color === void 0 && !italic && weight === 0) continue;
		if (name === "foreground") fg = color;
		else if (name === "background") bg = color;
		else styles[i] = style;
	}
	let htmlOpeners;
	return {
		name: theme.name,
		styles,
		fg,
		bg,
		table: usesDisplayP3 ? void 0 : table,
		get htmlOpeners() {
			if (!usesDisplayP3) return void 0;
			return htmlOpeners ??= themeHtmlOpeners(styles, fg, bg);
		}
	};
}
/**
* Pack theme colors and font styles for the Wasm HTML emitter.
* Escape attributes because CSS prefixes and defaults may contain quotes.
*/
function themeHtmlOpeners(styles, fg, bg) {
	const openers = [`<pre class="highlights" style="${escapeAttribute((bg === void 0 ? "" : `background-color:${bg};`) + (fg === void 0 ? "" : `color:${fg}`))}"><code>`];
	for (let i = 1; i < tokenTypes.length; i++) {
		const style = styles[i];
		const css = `color:${style?.color ?? "inherit"}` + (style?.italic === true ? ";font-style:italic" : "") + (style != null && style.weight !== 0 ? `;font-weight:${style.weight}` : "");
		openers.push(`<span style="${escapeAttribute(css)}">`);
	}
	return packHtmlOpeners(openers);
}
//#endregion
export { defaultCssVariablePrefix, escapeAttribute, isThemeColor, packHtmlOpeners, prepareTheme, resolveTheme, resolveThemeStyle, resolveThemeSyntax, themeBackground, themeForeground, themeTableBytes };

//# sourceMappingURL=theme.js.map