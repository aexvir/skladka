import tokenTypes from "./token-types.js";
import { escapeAttribute, packHtmlOpeners, prepareTheme } from "./theme.js";
//#region lib/tokens.ts
const enc = new TextEncoder();
/**
* Map each `$Token` slot to Shiki's `ThemedToken.type`.
*
* Values match TextMate's `StandardTokenType`: `0` other, `1` comment,
* `2` string, `3` regex. Bracket matching and editor heuristics ignore brackets
* in non-zero ranges.
*/
const standardTypes = new Uint8Array(tokenTypes.length);
for (let i = 0; i < tokenTypes.length; i++) {
	const name = tokenTypes[i];
	if (name === "string.regex") standardTypes[i] = 3;
	else if (name === "comment" || name.startsWith("comment.")) standardTypes[i] = 1;
	else if (name === "string" || name.startsWith("string.")) standardTypes[i] = 2;
}
const htmlStyleCache = /* @__PURE__ */ new WeakMap();
const blobCache = /* @__PURE__ */ new WeakMap();
let nextBlobId = 1;
const htmlOpenersCache = /* @__PURE__ */ new WeakMap();
const resolvedSetCache = /* @__PURE__ */ new WeakMap();
const preparedIds = /* @__PURE__ */ new WeakMap();
let nextPreparedId = 0;
/** A stable number for a prepared theme, assigned on first use. */
function preparedId(prepared) {
	let id = preparedIds.get(prepared);
	if (id === void 0) {
		id = nextPreparedId++;
		preparedIds.set(prepared, id);
	}
	return id;
}
/** Tag one prepared theme with its color key and role. */
function tagTheme({ name, styles, fg, bg, table }, color, role) {
	return {
		color,
		role,
		name,
		styles,
		fg,
		bg,
		table
	};
}
/**
* Convert Shiki-style options to a theme list.
* With `themes`, put the inline `defaultColor` theme first (`light` by
* default). Sort the remaining themes by name and use custom properties.
* `defaultColor: false` uses custom properties for all themes.
* `'light-dark()'` pairs the `light` and `dark` themes.
* Reuse cached list objects so their style caches remain available.
*/
function resolveOptionThemes(options) {
	const { cssVariablePrefix } = options;
	if (options.themes == null) return [tagTheme(prepareTheme(options.theme, cssVariablePrefix), null, "single")];
	const entries = Object.entries(options.themes).filter(([, t]) => t != null).sort(([a], [b]) => a < b ? -1 : 1);
	if (entries.length === 0) throw new TypeError("themes must not be empty");
	const defaultColor = options.defaultColor ?? "light";
	let roles;
	if (defaultColor === "light-dark()") {
		const light = entries.find(([key]) => key === "light");
		const dark = entries.find(([key]) => key === "dark");
		if (light === void 0 || dark === void 0) throw new TypeError("`themes` must contain `light` and `dark` when defaultColor is 'light-dark()'");
		entries.splice(0, entries.length, light, dark);
		roles = ["light-dark", "light-dark"];
	} else {
		if (defaultColor !== false) {
			const at = entries.findIndex(([key]) => key === defaultColor);
			if (at < 0) throw new TypeError(`\`themes\` must contain the defaultColor key \`${defaultColor}\``);
			entries.unshift(...entries.splice(at, 1));
		}
		roles = entries.map((_, i) => defaultColor !== false && i === 0 ? "default" : "variable");
	}
	const prepared = entries.map(([, theme]) => prepareTheme(theme, cssVariablePrefix));
	const key = JSON.stringify(entries.map(([color], i) => [
		preparedId(prepared[i]),
		roles[i],
		color
	]));
	let sets = resolvedSetCache.get(prepared[0]);
	const cached = sets?.get(key);
	if (cached !== void 0) return cached;
	const themes = entries.map(([color], i) => tagTheme(prepared[i], color, roles[i]));
	if (sets === void 0) {
		sets = /* @__PURE__ */ new Map();
		resolvedSetCache.set(prepared[0], sets);
	}
	sets.set(key, themes);
	if (sets.size > 128) {
		const oldest = sets.keys().next().value;
		if (oldest !== void 0) sets.delete(oldest);
	}
	return themes;
}
/** Convert an offset range and token id to a Shiki `ThemedToken`. */
function rangeToToken(code, start, end, hl, themes, cssVariablePrefix, offsetBase = 0) {
	const token = {
		content: code.slice(start, end),
		offset: start + offsetBase
	};
	if (themes[0].role === "single") {
		const { styles, fg } = themes[0];
		const style = styles[hl];
		token.color = style?.color ?? fg;
		let bits = 0;
		if (style?.italic === true) bits |= 1;
		if ((style?.weight ?? 0) >= 600) bits |= 2;
		token.fontStyle = bits;
	} else token.htmlStyle = themeHtmlStyle(themes, hl, cssVariablePrefix);
	const type = standardTypes[hl];
	if (type !== 0) token.type = type;
	return token;
}
function themeHtmlStyles(themes, cssVariablePrefix) {
	let prefixes = htmlStyleCache.get(themes);
	if (prefixes === void 0) {
		prefixes = /* @__PURE__ */ new Map();
		htmlStyleCache.set(themes, prefixes);
	}
	let slots = prefixes.get(cssVariablePrefix);
	if (slots === void 0) {
		slots = [];
		prefixes.set(cssVariablePrefix, slots);
	}
	return slots;
}
/**
* Build `htmlStyle` for token ID `hl` and a theme set.
* Use plain styles for the default theme and custom properties for the
* others, or combine colors with `light-dark()`.
* Cache by set, prefix, and token ID. All matching tokens and HTML tags
* share the map.
*/
function themeHtmlStyle(themes, hl, cssVariablePrefix) {
	const slots = themeHtmlStyles(themes, cssVariablePrefix);
	let htmlStyle = slots[hl];
	if (htmlStyle === void 0) {
		if (themes[0].role === "light-dark") htmlStyle = lightDarkStyle(themes, hl, cssVariablePrefix);
		else {
			htmlStyle = {};
			for (const { color, role, styles, fg } of themes) {
				const style = styles[hl];
				const plain = role === "default";
				htmlStyle[plain ? "color" : cssVariablePrefix + color] = style?.color ?? fg ?? "inherit";
				if (style?.italic === true) htmlStyle[plain ? "font-style" : `${cssVariablePrefix}${color}-font-style`] = "italic";
				if (style != null && style.weight !== 0) htmlStyle[plain ? "font-weight" : `${cssVariablePrefix}${color}-font-weight`] = String(style.weight);
			}
		}
		slots[hl] = htmlStyle;
	}
	return htmlStyle;
}
/**
* Pack a theme set for Wasm HTML mode 2. The emitter writes each theme's
* colors and font settings into spans. Layout (see `$multiRec` in emit.wat):
*
*     0   u8  slot count       1  u8  kind: 0 custom properties, 1 light-dark()
*     4   u32 tables offset    8  u32 span reserve    12  u32 prologue reserve
*     16  u32 set id
*     32  per slot, 16 bytes: u32 name offset, u32 name length, u32 inline
*     then one padded theme table per slot, then the name bytes
*
* Each slot name is an escaped custom property, such as `--hls-dark`.
* The inline `defaultColor` slot has no name. Reserves bound the size of
* span and `<pre>` openers, including property names supplied by the host.
* The unique set ID identifies this data in the opener cache.
*
* Cache by set and prefix. Return `undefined` if any theme lacks a packed
* table (Display P3 or CSS-variable colors). HTML output then uses
* `multiThemeHtmlOpeners`.
*/
function multiThemeBlob(themes, cssVariablePrefix) {
	let prefixes = blobCache.get(themes);
	if (prefixes === void 0) {
		prefixes = /* @__PURE__ */ new Map();
		blobCache.set(themes, prefixes);
	}
	const cached = prefixes.get(cssVariablePrefix);
	if (cached !== void 0) return cached ?? void 0;
	const count = themes.length;
	if (!(count <= 255 && themes.every((t) => t.table !== void 0))) {
		prefixes.set(cssVariablePrefix, null);
		return;
	}
	const names = themes.map((t) => t.role === "default" ? /* @__PURE__ */ new Uint8Array(0) : enc.encode(escapeAttribute(`${cssVariablePrefix}${t.color}`)));
	const tablesOffset = 32 + 16 * count;
	const namesOffset = tablesOffset + 384 * count;
	let spanReserve = 64;
	let prologueReserve = 128;
	let nameBytes = 0;
	for (const { length } of names) {
		spanReserve += 64 + 3 * length;
		prologueReserve += 48 + 2 * length;
		nameBytes += length;
	}
	const blob = new Uint8Array(namesOffset + nameBytes);
	const dv = new DataView(blob.buffer);
	blob[0] = count;
	blob[1] = themes[0].role === "light-dark" ? 1 : 0;
	dv.setUint32(4, tablesOffset, true);
	dv.setUint32(8, spanReserve, true);
	dv.setUint32(12, prologueReserve, true);
	dv.setUint32(16, nextBlobId++, true);
	let nameOffset = namesOffset;
	for (let i = 0; i < count; i++) {
		const slot = 32 + 16 * i;
		dv.setUint32(slot, nameOffset, true);
		dv.setUint32(slot + 4, names[i].length, true);
		dv.setUint32(slot + 8, themes[i].role === "default" ? 1 : 0, true);
		blob.set(themes[i].table, tablesOffset + 384 * i);
		blob.set(names[i], nameOffset);
		nameOffset += names[i].length;
	}
	prefixes.set(cssVariablePrefix, blob);
	return blob;
}
/**
* Build HTML openers for theme sets without packed RGBA tables.
* Spans use `htmlStyle`. The root uses `themeMeta`.
* Cache by set and prefix. Escape prefixes and theme keys for HTML attributes.
*/
function multiThemeHtmlOpeners(themes, cssVariablePrefix) {
	let prefixes = htmlOpenersCache.get(themes);
	if (prefixes === void 0) {
		prefixes = /* @__PURE__ */ new Map();
		htmlOpenersCache.set(themes, prefixes);
	}
	const cached = prefixes.get(cssVariablePrefix);
	if (cached !== void 0) return cached;
	const { fg, bg, rootStyle } = themeMeta(themes, cssVariablePrefix);
	const openers = [`<pre class="highlights" style="${escapeAttribute(rootStyle ?? `background-color:${bg};color:${fg}`)}"><code>`];
	for (let hl = 1; hl < tokenTypes.length; hl++) {
		const htmlStyle = themeHtmlStyle(themes, hl, cssVariablePrefix);
		let css = "";
		for (const property in htmlStyle) css += `${css === "" ? "" : ";"}${property}:${htmlStyle[property]}`;
		openers.push(`<span style="${escapeAttribute(css)}">`);
	}
	const blob = packHtmlOpeners(openers);
	prefixes.set(cssVariablePrefix, blob);
	return blob;
}
/**
* Build `htmlStyle` for `defaultColor: 'light-dark()'`.
* Combine both colors with CSS `light-dark()`. Use plain font settings
* when the themes agree, or per-theme custom properties when they differ.
*/
function lightDarkStyle(themes, hl, cssVariablePrefix) {
	const [light, dark] = themes;
	const a = light.styles[hl];
	const b = dark.styles[hl];
	const ac = a?.color ?? light.fg ?? "currentcolor";
	const bc = b?.color ?? dark.fg ?? "currentcolor";
	const htmlStyle = { color: ac === bc ? ac : `light-dark(${ac}, ${bc})` };
	const ai = a?.italic === true;
	const bi = b?.italic === true;
	if (ai && bi) htmlStyle["font-style"] = "italic";
	else {
		if (ai) htmlStyle[`${cssVariablePrefix}light-font-style`] = "italic";
		if (bi) htmlStyle[`${cssVariablePrefix}dark-font-style`] = "italic";
	}
	const aw = a?.weight ?? 0;
	const bw = b?.weight ?? 0;
	if (aw !== 0 && aw === bw) htmlStyle["font-weight"] = String(aw);
	else {
		if (aw !== 0) htmlStyle[`${cssVariablePrefix}light-font-weight`] = String(aw);
		if (bw !== 0) htmlStyle[`${cssVariablePrefix}dark-font-weight`] = String(bw);
	}
	return htmlStyle;
}
/** Convert UTF-16 token records with `0xffffffff` line markers to tokens. */
function lineRecordsToTokens(code, recs, themes, cssVariablePrefix, maxLineLength, offsetBase = 0) {
	const lines = [];
	const max = maxLineLength ?? 0;
	const { styles, fg, role } = themes[0];
	const htmlStyles = role === "single" ? void 0 : themeHtmlStyles(themes, cssVariablePrefix);
	let start = 0;
	for (let rec = 0;; rec += 2) {
		const first = rec;
		while (rec < recs.length && recs[rec + 1] !== 4294967295) rec += 2;
		const end = rec === first ? start : recs[rec - 2];
		let line;
		if (max > 0 && end - start >= max) line = [rangeToToken(code, start, end, 0, themes, cssVariablePrefix, offsetBase)];
		else {
			line = new Array((rec - first) / 2);
			let count = 0;
			for (let at = first; at < rec; at += 2) {
				const end = recs[at];
				if (end > start) {
					const hl = recs[at + 1];
					let token;
					if (htmlStyles === void 0) {
						const style = styles[hl];
						token = {
							content: code.slice(start, end),
							offset: start + offsetBase,
							color: style?.color ?? fg,
							fontStyle: (style?.italic === true ? 1 : 0) | ((style?.weight ?? 0) >= 600 ? 2 : 0)
						};
					} else token = {
						content: code.slice(start, end),
						offset: start + offsetBase,
						htmlStyle: htmlStyles[hl] ?? themeHtmlStyle(themes, hl, cssVariablePrefix)
					};
					const type = standardTypes[hl];
					if (type !== 0) token.type = type;
					line[count++] = token;
					start = end;
				}
			}
			line.length = count;
		}
		lines.push(line);
		if (rec === recs.length) break;
		start = recs[rec];
	}
	return lines;
}
/**
* Build `fg`, `bg`, `themeName`, and `rootStyle` for a Shiki `TokensResult`.
* With `themes`, `fg` and `bg` are CSS declaration lists. Each list starts
* with the default theme's plain color, then the other themes' properties.
* Set `rootStyle` only when no theme is inline. The `<pre>` style uses
* `rootStyle` if set, or `background-color:bg;color:fg` otherwise.
*/
function themeMeta(themes, cssVariablePrefix) {
	if (themes[0].role === "single") return {
		fg: themes[0].fg,
		bg: themes[0].bg,
		themeName: themes[0].name
	};
	const themeName = `highlights-themes ${themes.map((t) => t.name).join(" ")}`;
	if (themes[0].role === "light-dark") {
		const pair = (a, b, fallback) => {
			const x = a ?? fallback;
			const y = b ?? fallback;
			return x === y ? x : `light-dark(${x}, ${y})`;
		};
		return {
			fg: pair(themes[0].fg, themes[1].fg, "currentcolor"),
			bg: pair(themes[0].bg, themes[1].bg, "transparent"),
			themeName
		};
	}
	const fg = themes.map((t) => t.role === "default" ? t.fg ?? "inherit" : `${cssVariablePrefix}${t.color}:${t.fg ?? "inherit"}`).join(";");
	const bg = themes.map((t) => t.role === "default" ? t.bg ?? "inherit" : `${cssVariablePrefix}${t.color}-bg:${t.bg ?? "inherit"}`).join(";");
	return {
		fg,
		bg,
		themeName,
		rootStyle: themes[0].role === "default" ? void 0 : `${fg};${bg}`
	};
}
//#endregion
export { lineRecordsToTokens, multiThemeBlob, multiThemeHtmlOpeners, rangeToToken, resolveOptionThemes, standardTypes, themeHtmlStyle, themeMeta };

//# sourceMappingURL=tokens.js.map