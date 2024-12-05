// Package components provides reusable UI components for the skladka frontend.
// Each component is implemented as a [templ] template and follows the principles
// of component-based design, encapsulating both markup and logic.
//
// The components in this package are designed to be:
//   - Self-contained: Each component manages its own state and behavior
//   - Reusable: Components can be used across different views
//   - Composable: Components can be combined to build more complex UIs
//   - Styled: Components use TailwindCSS for consistent styling
//
// # Available Components
//   - Editor: Monaco-based code editor with syntax highlighting
//   - Entry: Individual paste entry display component
//   - Input: Form input fields with consistent styling
//   - Nav: Navigation bar component
//   - Palette: Color scheme selection component
//   - Sidebar: Collapsible sidebar for navigation
//   - Toggle: Interactive toggle switch component
//
// # Example Usage
//
//	func Page(title string) templ.Component {
//		return components.Base(
//			components.Nav(title),
//			components.Editor(
//				components.EditorOptions{
//					Language: "go",
//					ReadOnly: false,
//				},
//			),
//		)
//	}
//
// Components are designed to work seamlessly with the [templ] templating engine
// and [TailwindCSS]. They provide a consistent look and feel across the application
// while maintaining type safety and reusability.
//
// [templ]: https://github.com/a-h/templ
// [TailwindCSS]: https://tailwindcss.com
package components
