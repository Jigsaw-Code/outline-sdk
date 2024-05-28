module github.com/Jigsaw-Code/outline-sdk/x/psiphon

go 1.20

// Use our non-functional stub implementation, instead of the official GPL one.
// Actual users will have to depend on the official GPL code instead.
replace github.com/Psiphon-Labs/psiphon-tunnel-core => ./stub

require (
	github.com/Jigsaw-Code/outline-sdk v0.0.16
	// Use github.com/Psiphon-Labs/psiphon-tunnel-core@staging-client as per
	// https://github.com/Psiphon-Labs/psiphon-tunnel-core/?tab=readme-ov-file#using-psiphon-with-go-modules
	github.com/Psiphon-Labs/psiphon-tunnel-core v1.0.11-0.20240522172529-8fcc4b9a51cf
)