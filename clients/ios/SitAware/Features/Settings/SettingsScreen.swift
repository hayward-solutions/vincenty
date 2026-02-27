import SwiftUI

/// Settings navigation — sidebar with Account sections + Server admin sections.
///
/// Mirrors the web client's `settings/account/layout.tsx` sidebar:
/// - General (profile, avatar, marker)
/// - Security (password, MFA)
/// - Devices
/// - Activity
/// - Groups
///
/// Admin-only section:
/// - Users, Groups, Map Config, Security Policy, Audit Logs
struct SettingsScreen: View {
    @Environment(AuthManager.self) private var auth

    var body: some View {
        NavigationStack {
            List {
                // MARK: - Account Settings
                Section("Account") {
                    NavigationLink {
                        ProfileView()
                    } label: {
                        Label("General", systemImage: "person.crop.circle")
                    }

                    NavigationLink {
                        SecurityView()
                    } label: {
                        Label("Security", systemImage: "lock.shield")
                    }

                    NavigationLink {
                        DevicesView()
                    } label: {
                        Label("Devices", systemImage: "iphone")
                    }

                    NavigationLink {
                        ActivityView()
                    } label: {
                        Label("Activity", systemImage: "clock.arrow.circlepath")
                    }

                    NavigationLink {
                        GroupsView()
                    } label: {
                        Label("Groups", systemImage: "person.3")
                    }

                    NavigationLink {
                        SystemLogView()
                    } label: {
                        Label("System Log", systemImage: "list.bullet.rectangle")
                    }
                }

                // MARK: - Server Settings (Admin only)
                if auth.isAdmin {
                    Section("Server") {
                        NavigationLink {
                            AdminUsersView()
                        } label: {
                            Label("Users", systemImage: "person.2")
                        }

                        NavigationLink {
                            AdminGroupsView()
                        } label: {
                            Label("Groups", systemImage: "person.3.fill")
                        }

                        NavigationLink {
                            AdminMapConfigView()
                        } label: {
                            Label("Map Configuration", systemImage: "map")
                        }

                        NavigationLink {
                            AdminSecurityPolicyView()
                        } label: {
                            Label("Security Policy", systemImage: "shield.checkered")
                        }

                        NavigationLink {
                            AdminAuditLogsView()
                        } label: {
                            Label("Audit Logs", systemImage: "doc.text.magnifyingglass")
                        }
                    }
                }

                // MARK: - Logout
                Section {
                    Button(role: .destructive) {
                        Task { await auth.logout() }
                    } label: {
                        Label("Sign Out", systemImage: "rectangle.portrait.and.arrow.right")
                    }
                }
            }
            .navigationTitle("Settings")
        }
    }
}
