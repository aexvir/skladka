// Package views provides the page templates for different sections of the skladka frontend.
// Each view is implemented as a [templ] template and represents a complete page or
// major section of the application.
//
// The views in this package are responsible for:
//   - Composing components into complete pages
//   - Implementing page-specific logic and interactions
//   - Managing the layout and structure of each section
//   - Handling data display and user input
//
// # Available Views
//   - Archive: Displays a list of all public pastes
//   - Creation: Form for creating new pastes
//   - Document: Displays a single paste with its content and metadata
//
// # Example Usage
//
//	func renderCreationPage(w http.ResponseWriter, r *http.Request) {
//		layouts.Base(
//			views.Creation("New Paste"),
//		).Render(r.Context(), w)
//	}
//
// Views are designed to work with the [layouts] package for consistent page structure
// and the [components] package for reusable UI elements. Each view is type-safe and
// compiled at build time by the [templ] engine.
package views
