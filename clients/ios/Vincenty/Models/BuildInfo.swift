import Foundation

enum BuildInfo {
    /// The app's version string from Info.plist
    static var version: String {
        Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "Unknown"
    }
    
    /// The app's build number from Info.plist
    static var buildNumber: String {
        Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "Unknown"
    }
    
    /// Combined version and build number (e.g., "1.0.0 (42)")
    static var fullVersion: String {
        "\(version) (\(buildNumber))"
    }
}
