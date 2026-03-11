import SwiftUI

/// Account security settings — change password + MFA management.
///
/// Mirrors the web client's `settings/account/security/page.tsx`:
/// - Change password form (current + new + confirm)
/// - MFA method list (TOTP + WebAuthn + recovery codes) — delegated to `MFAMethodsView`
struct SecurityView: View {
    @Environment(AuthManager.self) private var auth

    // Password form
    @State private var currentPassword = ""
    @State private var newPassword = ""
    @State private var confirmPassword = ""
    @State private var isChangingPassword = false
    @State private var passwordError: String?
    @State private var passwordSuccess: String?

    private let api = APIClient.shared

    var body: some View {
        Form {
            // MARK: - Change Password
            Section("Password") {
                SecureField("Current Password", text: $currentPassword)
                    .textContentType(.password)

                SecureField("New Password (min 8 characters)", text: $newPassword)
                    .textContentType(.newPassword)

                SecureField("Confirm New Password", text: $confirmPassword)
                    .textContentType(.newPassword)

                if let error = passwordError {
                    Text(error)
                        .foregroundStyle(.red)
                        .font(.caption)
                }

                if let success = passwordSuccess {
                    Text(success)
                        .foregroundStyle(.green)
                        .font(.caption)
                }

                Button {
                    Task { await changePassword() }
                } label: {
                    if isChangingPassword {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                    } else {
                        Text("Change Password")
                            .frame(maxWidth: .infinity)
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(isChangingPassword || !isPasswordFormValid)
            }

            // MARK: - Multi-Factor Authentication
            Section("Multi-Factor Authentication") {
                MFAMethodsView()
            }

            // MARK: - API Tokens
            Section("API Tokens") {
                Text("API token management coming soon.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
        }
        .navigationTitle("Security")
    }

    // MARK: - Validation

    private var isPasswordFormValid: Bool {
        !currentPassword.isEmpty
            && newPassword.count >= 8
            && newPassword == confirmPassword
    }

    // MARK: - Actions

    private func changePassword() async {
        passwordError = nil
        passwordSuccess = nil
        isChangingPassword = true

        guard newPassword == confirmPassword else {
            passwordError = "Passwords do not match"
            isChangingPassword = false
            return
        }

        guard newPassword.count >= 8 else {
            passwordError = "Password must be at least 8 characters"
            isChangingPassword = false
            return
        }

        do {
            let body = ChangePasswordRequest(
                currentPassword: currentPassword,
                newPassword: newPassword)
            let _: EmptyResponse = try await api.put(Endpoints.usersMePassword, body: body)

            currentPassword = ""
            newPassword = ""
            confirmPassword = ""
            passwordSuccess = "Password changed successfully"
        } catch let error as APIError {
            passwordError = error.message
        } catch {
            passwordError = error.localizedDescription
        }

        isChangingPassword = false
    }
}
