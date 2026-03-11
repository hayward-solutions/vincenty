import SwiftUI

/// Admin security policy — server-wide security settings.
///
/// Mirrors the web client's `settings/server/security/page.tsx`:
/// - MFA enforcement toggle (require MFA for all users)
struct AdminSecurityPolicyView: View {
    @State private var mfaRequired = false
    @State private var isLoading = true
    @State private var isSaving = false
    @State private var errorMessage: String?
    @State private var showConfirmEnable = false

    private let api = APIClient.shared

    var body: some View {
        List {
            // MARK: - MFA Policy
            Section {
                VStack(alignment: .leading, spacing: 12) {
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Require MFA for all users")
                                .font(.subheadline.weight(.medium))
                            Text("When enabled, users without MFA will be blocked until they configure it.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }

                        Spacer()

                        if isLoading {
                            ProgressView()
                        } else {
                            Toggle("", isOn: .init(
                                get: { mfaRequired },
                                set: { newValue in
                                    if newValue {
                                        showConfirmEnable = true
                                    } else {
                                        Task { await toggleMFA(enabled: false) }
                                    }
                                }
                            ))
                            .labelsHidden()
                        }
                    }

                    if mfaRequired {
                        HStack(spacing: 8) {
                            Image(systemName: "exclamationmark.triangle.fill")
                                .foregroundStyle(.orange)
                            Text("Users without MFA will be blocked from accessing the system until they configure it.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        .padding(10)
                        .background(Color.orange.opacity(0.1))
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                    }
                }
            } header: {
                Text("Multi-Factor Authentication")
            }

            if let error = errorMessage {
                Section {
                    Text(error)
                        .foregroundStyle(.red)
                        .font(.caption)
                }
            }
        }
        .navigationTitle("Security Policy")
        .task { await loadSettings() }
        .alert("Require MFA", isPresented: $showConfirmEnable) {
            Button("Cancel", role: .cancel) {}
            Button("Enable") {
                Task { await toggleMFA(enabled: true) }
            }
        } message: {
            Text("Enabling this will block users who haven't configured MFA. Are you sure?")
        }
    }

    // MARK: - API Actions

    private func loadSettings() async {
        isLoading = true
        do {
            let settings: ServerSettings = try await api.get(Endpoints.serverSettings)
            mfaRequired = settings.mfaRequired
        } catch {
            errorMessage = "Failed to load settings"
        }
        isLoading = false
    }

    private func toggleMFA(enabled: Bool) async {
        isSaving = true
        errorMessage = nil

        do {
            struct SettingsBody: Encodable {
                let mfaRequired: Bool
            }
            let settings: ServerSettings = try await api.put(
                Endpoints.serverSettings,
                body: SettingsBody(mfaRequired: enabled))
            mfaRequired = settings.mfaRequired
        } catch {
            errorMessage = "Failed to update security policy"
        }

        isSaving = false
    }
}
