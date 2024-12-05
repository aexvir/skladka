// Package frontend provides the web interface for the skladka service built on top of [templ]
// templating engine and [TailwindCSS] for styling. It offers a modern, type-safe, and
// component-based UI for creating, viewing, and managing pastes.
//
// The package is structured into several subpackages:
//   - layouts: Base layout templates that define the overall page structure
//   - views: Individual page templates for different sections of the application
//   - components: Reusable UI components that encapsulate markup and logic
//   - icons: SVG icons used throughout the UI, implemented as templ components
//
// # Features
//   - Modern UI built with TailwindCSS for responsive and maintainable styling
//   - Monaco editor integration for syntax-highlighted code editing and viewing
//   - Type-safe templates compiled at build time using templ
//   - Static asset serving from an embedded filesystem
//   - Chi-based routing for clean URL structure
//
// # example Usage
//
//	router := chi.NewRouter()
//	storage := paste.NewStorage()
//
//	// Mount the frontend router
//	router.Mount("/", frontend.DashboardRouter(storage))
//
//	// Start the server
//	http.ListenAndServe(":8080", router)
//
// The frontend package automatically handles static asset serving, page routing,
// and template rendering. All templates are type-safe and compiled at build time,
// providing compile-time validation and IDE support.
//
// [templ]: https://github.com/a-h/templ
// [TailwindCSS]: https://tailwindcss.com
package frontend
