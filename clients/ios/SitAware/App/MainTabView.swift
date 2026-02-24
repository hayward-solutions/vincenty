import SwiftUI

/// Primary tab navigation for the authenticated app.
/// Maps to the web client's sidebar navigation.
struct MainTabView: View {
    @Environment(AuthManager.self) private var auth
    @State private var selectedTab: Tab = .map

    enum Tab: String {
        case dashboard
        case map
        case messages
        case settings
    }

    var body: some View {
        TabView(selection: $selectedTab) {
            SwiftUI.Tab("Dashboard", systemImage: "square.grid.2x2", value: .dashboard) {
                DashboardView()
            }

            SwiftUI.Tab("Map", systemImage: "map", value: .map) {
                MapScreen()
            }

            SwiftUI.Tab("Messages", systemImage: "message", value: .messages) {
                MessagesScreen()
            }

            SwiftUI.Tab("Settings", systemImage: "gear", value: .settings) {
                SettingsScreen()
            }
        }
    }
}

// MARK: - Dashboard

/// Dashboard overview — matches web's `/dashboard`.
struct DashboardView: View {
    @Environment(AuthManager.self) private var auth

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                    // User card
                    if let user = auth.user {
                        VStack(spacing: 12) {
                            Image(systemName: "person.circle.fill")
                                .font(.system(size: 64))
                                .foregroundStyle(.secondary)

                            Text(user.displayName.isEmpty ? user.username : user.displayName)
                                .font(.title2.bold())

                            Text("@\(user.username)")
                                .font(.subheadline)
                                .foregroundStyle(.secondary)

                            HStack(spacing: 8) {
                                Badge(
                                    text: user.isAdmin ? "Admin" : "User",
                                    color: user.isAdmin ? .blue : .secondary)

                                if user.mfaEnabled {
                                    Badge(text: "MFA Enabled", color: .green)
                                }
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(.regularMaterial)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                    }
                }
                .padding()
            }
            .navigationTitle("Dashboard")
        }
    }
}

// MARK: - Badge Helper

private struct Badge: View {
    let text: String
    let color: Color

    var body: some View {
        Text(text)
            .font(.caption.weight(.medium))
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(color.opacity(0.15))
            .foregroundStyle(color)
            .clipShape(Capsule())
    }
}

// MARK: - Placeholder (removed — Settings implemented)
