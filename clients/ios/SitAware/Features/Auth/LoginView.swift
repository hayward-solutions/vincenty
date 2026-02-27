import SwiftUI

/// Login screen with username/password form and passkey button.
/// Mirrors the web client's `login/page.tsx`.
struct LoginView: View {
    @Environment(AuthManager.self) private var auth

    @State private var username = ""
    @State private var password = ""
    @State private var isLoading = false
    @State private var error: String?
    @State private var mfaChallenge: MFAChallengeResponse?

    var body: some View {
        if let challenge = mfaChallenge {
            MFAChallengeView(
                challenge: challenge,
                onSuccess: { response in
                    Task { await auth.completeMFALogin(response) }
                    mfaChallenge = nil
                },
                onCancel: { mfaChallenge = nil }
            )
        } else {
            loginForm
        }
    }

    private var loginForm: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Spacer()

                // Logo
                VStack(spacing: 8) {
                    Image(systemName: "antenna.radiowaves.left.and.right")
                        .font(.system(size: 48))
                        .foregroundStyle(.tint)
                        .accessibilityHidden(true)

                    Text("SitAware")
                        .font(.title.bold())
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel("SitAware")

                // Form
                VStack(spacing: 16) {
                    TextField("Username", text: $username)
                        .textFieldStyle(.roundedBorder)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .textContentType(.username)
                        .submitLabel(.next)

                    SecureField("Password", text: $password)
                        .textFieldStyle(.roundedBorder)
                        .textContentType(.password)
                        .submitLabel(.go)
                        .onSubmit { Task { await login() } }
                }
                .padding(.horizontal)

                if let error {
                    Text(error)
                        .font(.callout)
                        .foregroundStyle(.red)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal)
                        .accessibilityLabel("Error: \(error)")
                }

                VStack(spacing: 12) {
                    Button(action: { Task { await login() } }) {
                        if isLoading {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                        } else {
                            Text("Sign In")
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.large)
                    .disabled(username.isEmpty || password.isEmpty || isLoading)

                    // Passkey login button
                    Button(action: { Task { await passkeyLogin() } }) {
                        Label("Sign in with Passkey", systemImage: "person.badge.key")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.large)
                }
                .padding(.horizontal)

                Spacer()

                // Server URL indicator
                if let serverURL = KeychainStore.shared.serverURL {
                    HStack(spacing: 4) {
                        Image(systemName: "server.rack")
                            .font(.caption2)
                        Text(serverURL)
                            .font(.caption2)
                    }
                    .foregroundStyle(.secondary)
                    .padding(.bottom, 8)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Connected to server \(serverURL)")
                }
            }
        }
    }

    private func login() async {
        guard !username.isEmpty, !password.isEmpty else { return }

        isLoading = true
        error = nil

        do {
            let result = try await auth.login(username: username, password: password)

            switch result {
            case .authenticated:
                // Auth state updated in AuthManager — UI will react
                break
            case .mfaRequired(let challenge):
                mfaChallenge = challenge
            }
        } catch let apiError as APIError {
            error = apiError.message
        } catch {
            self.error = "An unexpected error occurred."
        }

        isLoading = false
    }

    private func passkeyLogin() async {
        // Passkey/WebAuthn login will be implemented in Phase 4 detail
        // using ASAuthorizationPlatformPublicKeyCredentialProvider
        error = "Passkey login not yet implemented."
    }
}
