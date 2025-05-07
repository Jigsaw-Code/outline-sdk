// TODO: install swift-tools
import PackageDescription

let package = Package(
    name: "MobileProxy",
    platforms: [
        .iOS(.v11)
    ],
    products: [
        .library(
            name: "Mobileproxy",
            targets: ["MobileproxyWrapper"]
        ),
    ],
    dependencies: [],
    targets: [
        .binaryTarget(
            name: "MobileproxyBinary",
            url: "https://github.com/Jigsaw-Code/outline-sdk/releases/download/x%2Fv0.0.3/mobileproxy.xcframework.zip", // << VERSIONED URL
            checksum: "TODO"
        ),
        .target(
            name: "MobileproxyWrapper",
            dependencies: ["MobileproxyBinary"],
            path: "Sources/MobileproxyWrapper",
            publicHeadersPath: "."
        )
    ]
)
