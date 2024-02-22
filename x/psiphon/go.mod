module github.com/Jigsaw-Code/outline-sdk/x/psiphon

go 1.20

// Use our non-functional stub implementation, instead of the official GPL one.
// Actual users will have to depend on the official GPL code instead.
replace github.com/Psiphon-Labs/psiphon-tunnel-core => ./stub

require (
	github.com/Jigsaw-Code/outline-sdk v0.0.13
	github.com/Psiphon-Labs/psiphon-tunnel-core v1.0.11-0.20231206204740-a8e5fc0cf6c7
)
