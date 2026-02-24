import SwiftUI

/// MFA challenge screen shown after login when MFA is required.
/// Mirrors the web client's `mfa-challenge.tsx`.
struct MFAChallengeView: View {
    let challenge: MFAChallengeResponse
    let onSuccess: (AuthResponse) -> Void
    let onCancel: () -> Void

    @State private var activeMethod: String
    @State private var code = ""
    @State private var isLoading = false
    @State private var error: String?

    private let api = APIClient.shared

    init(
        challenge: MFAChallengeResponse,
        onSuccess: @escaping (AuthResponse) -> Void,
        onCancel: @escaping () -> Void
    ) {
        self.challenge = challenge
        self.onSuccess = onSuccess
        self.onCancel = onCancel
        // Default to first available method
        self._activeMethod = State(initialValue: challenge.methods.first ?? "totp")
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Spacer()

                // Header
                VStack(spacing: 8) {
                    Image(systemName: "lock.shield")
                        .font(.system(size: 48))
                        .foregroundStyle(.accent)

                    Text("Two-Factor Authentication")
                        .font(.title2.bold())

                    Text("Verify your identity to continue.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                // Method selector (if multiple methods available)
                if challenge.methods.count > 1 {
                    Picker("Method", selection: $activeMethod) {
                        if challenge.methods.contains("totp") {
                            Text("Authenticator").tag("totp")
                        }
                        if challenge.methods.contains("webauthn") {
                            Text("Security Key").tag("webauthn")
                        }
                        if challenge.methods.contains("recovery") {
                            Text("Recovery").tag("recovery")
                        }
                    }
                    .pickerStyle(.segmented)
                    .padding(.horizontal)
                    .onChange(of: activeMethod) { _, _ in
                        code = ""
                        error = nil
                    }
                }

                // Method-specific input
                VStack(spacing: 16) {
                    switch activeMethod {
                    case "totp":
                        totpPanel
                    case "webauthn":
                        webauthnPanel
                    case "recovery":
                        recoveryPanel
                    default:
                        Text("Unknown method")
                    }
                }
                .padding(.horizontal)

                if let error {
                    Text(error)
                        .font(.callout)
                        .foregroundStyle(.red)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal)
                }

                Spacer()

                Button("Cancel", action: onCancel)
                    .foregroundStyle(.secondary)
                    .padding(.bottom)
            }
        }
    }

    // MARK: - TOTP

    private var totpPanel: some View {
        VStack(spacing: 16) {
            Text("Enter the 6-digit code from your authenticator app.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)

            TextField("000000", text: $code)
                .textFieldStyle(.roundedBorder)
                .keyboardType(.numberPad)
                .multilineTextAlignment(.center)
                .font(.title2.monospaced())
                .onChange(of: code) { _, newValue in
                    code = String(newValue.filter(\.isNumber).prefix(6))
                }

            Button(action: { Task { await verifyTOTP() } }) {
                if isLoading {
                    ProgressView().frame(maxWidth: .infinity)
                } else {
                    Text("Verify").frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
            .disabled(code.count != 6 || isLoading)
        }
    }

    // MARK: - WebAuthn

    private var webauthnPanel: some View {
        VStack(spacing: 16) {
            Text("Use your security key to verify your identity.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)

            Button(action: { Task { await verifyWebAuthn() } }) {
                if isLoading {
                    ProgressView().frame(maxWidth: .infinity)
                } else {
                    Label("Use Security Key", systemImage: "key")
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
            .disabled(isLoading)
        }
    }

    // MARK: - Recovery

    private var recoveryPanel: some View {
        VStack(spacing: 16) {
            Text("Enter one of your recovery codes.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)

            TextField("xxxx-xxxx", text: $code)
                .textFieldStyle(.roundedBorder)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .multilineTextAlignment(.center)
                .font(.title3.monospaced())

            Button(action: { Task { await verifyRecovery() } }) {
                if isLoading {
                    ProgressView().frame(maxWidth: .infinity)
                } else {
                    Text("Verify").frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
            .disabled(code.isEmpty || isLoading)
        }
    }

    // MARK: - Actions

    private func verifyTOTP() async {
        isLoading = true
        error = nil
        do {
            struct Body: Encodable {
                let mfaToken: String
                let code: String
            }
            let response: AuthResponse = try await api.post(
                Endpoints.mfaTOTP,
                body: Body(mfaToken: challenge.mfaToken, code: code))
            onSuccess(response)
        } catch let apiError as APIError {
            error = apiError.message
        } catch {
            self.error = "Verification failed."
        }
        isLoading = false
    }

    private func verifyWebAuthn() async {
        // WebAuthn assertion via ASAuthorization — full implementation in Phase 12
        error = "WebAuthn MFA challenge will use ASAuthorizationPlatformPublicKeyCredentialProvider."
    }

    private func verifyRecovery() async {
        isLoading = true
        error = nil
        do {
            struct Body: Encodable {
                let mfaToken: String
                let code: String
            }
            let response: AuthResponse = try await api.post(
                Endpoints.mfaRecovery,
                body: Body(mfaToken: challenge.mfaToken, code: code))
            onSuccess(response)
        } catch let apiError as APIError {
            error = apiError.message
        } catch {
            self.error = "Verification failed."
        }
        isLoading = false
    }
}
