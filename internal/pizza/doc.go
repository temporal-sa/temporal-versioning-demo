// Package pizza contains the Pizza Tracker demo's Temporal workflow and
// activities: an order moving through Received, Cooking, (Quality check),
// Out for delivery / Drone delivery, and Delivered.
//
// The workflow declares the Pinned versioning behavior and exposes a getState
// Query returning the order's current step and the worker version handling it.
package pizza
