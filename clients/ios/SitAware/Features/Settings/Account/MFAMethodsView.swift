import SwiftUI

/// MFA method management — list, add, and remove MFA methods.
///
/// Mirrors the web client's `mfa-method-list.tsx`:
/// - Lists enrolled TOTP and WebAuthn methods
/// - Add Authenticator App (TOTP setup flow)
/// - Add Security Key (WebAuthn — placeholder since iOS uses ASAuthorization)
/// - Regenerate recovery codes
struct MFAMethodsView: View {
    @State private var methods: [MFAMethod] = []
    @State private var isLoading = true
    @State private var errorMessage: String?

    // TOTP setup
    @State private var showTOTPSetup = false
    @State private var totpSetup: TOTPSetupResponse?
    @State private var totpName = "Authenticator App"
    @State private var totpCode = ""
    @State private var totpSetupStep: TOTPSetupStep = .name
    @State private var recoveryCodes: [String]?
    @State private var isTOTPSetupInProgress = false

    // Recovery codes
    @State private var showRegenerateRecovery = false
    @State private var isRegenerating = false
    @State private var regeneratedCodes: [String]?

    // Remove confirmation
    @State private var methodToRemove: MFAMethod?
    @State private var showRemoveConfirmation = false
    @State private var isRemoving = false

    private let api = APIClient.shared

    enum TOTPSetupStep {
        case name, scan, verify, recovery
    }

    var body: some View {
        Group {
            if isLoading {
                ProgressView()
            } else {
                methodsList
            }
        }
        .task { await loadMethods() }
        .sheet(isPresented: $showTOTPSetup) { totpSetupSheet }
        .sheet(isPresented: $showRegenerateRecovery) { regenerateRecoverySheet }
        .alert("Remove MFA Method", isPresented: $showRemoveConfirmation) {
            Button("Cancel", role: .cancel) {}
            Button("Remove", role: .destructive) {
                if let method = methodToRemove {
                    Task { await removeMethod(method) }
                }
            }
        } message: {
            Text("Are you sure you want to remove \"\(methodToRemove?.name ?? "")\"? This cannot be undone.")
        }
    }

    // MARK: - Methods List

    @ViewBuilder
    private var methodsList: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Authenticator Apps (TOTP)
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Label("Authenticator App", systemImage: "apps.iphone")
                        .font(.subheadline.weight(.medium))
                    Spacer()
                    Button("Add") { beginTOTPSetup() }
                        .font(.caption)
                }

                let totpMethods = methods.filter { $0.type == .totp }
                if totpMethods.isEmpty {
                    Text("No authenticator app configured")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(totpMethods) { method in
                        methodRow(method)
                    }
                }
            }

            Divider()

            // Security Keys (WebAuthn)
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Label("Security Keys & Passkeys", systemImage: "key.fill")
                        .font(.subheadline.weight(.medium))
                    Spacer()
                    Button("Add") {
                        // WebAuthn registration on iOS uses ASAuthorization
                        // TODO: Implement WebAuthn registration via ASAuthorizationPlatformPublicKeyCredentialProvider
                    }
                    .font(.caption)
                }

                let webauthnMethods = methods.filter { $0.type == .webauthn }
                if webauthnMethods.isEmpty {
                    Text("No security keys registered")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(webauthnMethods) { method in
                        methodRow(method)
                    }
                }
            }

            // Recovery codes — only if any MFA method exists
            if !methods.isEmpty {
                Divider()

                HStack {
                    Label("Recovery Codes", systemImage: "doc.text")
                        .font(.subheadline.weight(.medium))
                    Spacer()
                    Button("Regenerate") { showRegenerateRecovery = true }
                        .font(.caption)
                }
            }
        }

        if let error = errorMessage {
            Text(error)
                .foregroundStyle(.red)
                .font(.caption)
        }
    }

    // MARK: - Method Row

    @ViewBuilder
    private func methodRow(_ method: MFAMethod) -> some View {
        HStack(spacing: 8) {
            Image(systemName: method.type == .totp ? "apps.iphone" : "key.fill")
                .font(.caption)
                .foregroundStyle(.secondary)

            VStack(alignment: .leading, spacing: 2) {
                Text(method.name)
                    .font(.subheadline)
                HStack(spacing: 4) {
                    Text("Added \(formatDate(method.createdAt))")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    if let lastUsed = method.lastUsedAt {
                        Text("Last used \(formatDate(lastUsed))")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Spacer()

            Button(role: .destructive) {
                methodToRemove = method
                showRemoveConfirmation = true
            } label: {
                Image(systemName: "trash")
                    .font(.caption)
            }
        }
        .padding(.vertical, 4)
    }

    // MARK: - TOTP Setup Sheet

    @ViewBuilder
    private var totpSetupSheet: some View {
        NavigationStack {
            Group {
                switch totpSetupStep {
                case .name:
                    totpNameStep
                case .scan:
                    totpScanStep
                case .verify:
                    totpVerifyStep
                case .recovery:
                    recoveryCodesStep
                }
            }
            .navigationTitle("Set Up Authenticator")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    if totpSetupStep != .recovery {
                        Button("Cancel") {
                            showTOTPSetup = false
                            resetTOTPSetup()
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var totpNameStep: some View {
        Form {
            Section("Name") {
                TextField("Authenticator name", text: $totpName)
            }

            Section {
                Button {
                    Task { await startTOTPSetup() }
                } label: {
                    if isTOTPSetupInProgress {
                        ProgressView().frame(maxWidth: .infinity)
                    } else {
                        Text("Continue").frame(maxWidth: .infinity)
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(totpName.trimmingCharacters(in: .whitespaces).isEmpty || isTOTPSetupInProgress)
            }
        }
    }

    @ViewBuilder
    private var totpScanStep: some View {
        VStack(spacing: 20) {
            if let setup = totpSetup {
                Text("Scan this QR code with your authenticator app")
                    .font(.subheadline)
                    .multilineTextAlignment(.center)
                    .padding(.top, 24)

                // QR code placeholder — on real iOS, use CoreImage CIFilter
                QRCodeView(uri: setup.uri)
                    .frame(width: 200, height: 200)

                // Manual entry
                VStack(spacing: 4) {
                    Text("Or enter this key manually:")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text(setup.secret)
                        .font(.system(.caption, design: .monospaced))
                        .textSelection(.enabled)
                        .padding(8)
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 6))
                }

                Button("I've scanned it") {
                    totpSetupStep = .verify
                }
                .buttonStyle(.borderedProminent)
            }

            Spacer()
        }
        .padding()
    }

    @ViewBuilder
    private var totpVerifyStep: some View {
        Form {
            Section("Enter the 6-digit code from your authenticator app") {
                TextField("000000", text: $totpCode)
                    .keyboardType(.numberPad)
                    .font(.system(.title2, design: .monospaced))
                    .multilineTextAlignment(.center)
            }

            if let error = errorMessage {
                Section {
                    Text(error)
                        .foregroundStyle(.red)
                        .font(.caption)
                }
            }

            Section {
                Button {
                    Task { await verifyTOTP() }
                } label: {
                    if isTOTPSetupInProgress {
                        ProgressView().frame(maxWidth: .infinity)
                    } else {
                        Text("Verify").frame(maxWidth: .infinity)
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(totpCode.count != 6 || isTOTPSetupInProgress)
            }
        }
    }

    @ViewBuilder
    private var recoveryCodesStep: some View {
        RecoveryCodesView(
            codes: recoveryCodes ?? [],
            onDone: {
                showTOTPSetup = false
                resetTOTPSetup()
                Task { await loadMethods() }
            }
        )
    }

    // MARK: - Regenerate Recovery Sheet

    @ViewBuilder
    private var regenerateRecoverySheet: some View {
        NavigationStack {
            if let codes = regeneratedCodes {
                RecoveryCodesView(
                    codes: codes,
                    onDone: {
                        showRegenerateRecovery = false
                        regeneratedCodes = nil
                    }
                )
                .navigationTitle("Recovery Codes")
                .navigationBarTitleDisplayMode(.inline)
            } else {
                VStack(spacing: 20) {
                    Image(systemName: "exclamationmark.triangle")
                        .font(.largeTitle)
                        .foregroundStyle(.orange)

                    Text("Regenerate Recovery Codes")
                        .font(.headline)

                    Text("This will invalidate all existing recovery codes. Make sure to save the new codes in a safe place.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal)

                    Button(role: .destructive) {
                        Task { await regenerateRecoveryCodes() }
                    } label: {
                        if isRegenerating {
                            ProgressView().frame(maxWidth: .infinity)
                        } else {
                            Text("Regenerate Codes").frame(maxWidth: .infinity)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.red)
                    .disabled(isRegenerating)
                    .padding(.horizontal)

                    Spacer()
                }
                .padding(.top, 32)
                .navigationTitle("Recovery Codes")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { showRegenerateRecovery = false }
                    }
                }
            }
        }
    }

    // MARK: - API Actions

    private func loadMethods() async {
        isLoading = true
        do {
            methods = try await api.get(Endpoints.usersMeMFAMethods)
        } catch {
            errorMessage = "Failed to load MFA methods"
        }
        isLoading = false
    }

    private func beginTOTPSetup() {
        totpName = "Authenticator App"
        totpSetupStep = .name
        totpCode = ""
        totpSetup = nil
        recoveryCodes = nil
        errorMessage = nil
        showTOTPSetup = true
    }

    private func startTOTPSetup() async {
        isTOTPSetupInProgress = true
        errorMessage = nil

        do {
            struct SetupBody: Encodable {
                let name: String
            }
            totpSetup = try await api.post(
                Endpoints.usersMeMFATOTPSetup,
                body: SetupBody(name: totpName))
            totpSetupStep = .scan
        } catch {
            errorMessage = "Failed to start TOTP setup: \(error.localizedDescription)"
        }

        isTOTPSetupInProgress = false
    }

    private func verifyTOTP() async {
        isTOTPSetupInProgress = true
        errorMessage = nil

        guard let setup = totpSetup else { return }

        do {
            struct VerifyBody: Encodable {
                let methodId: String
                let code: String
            }
            let response: TOTPVerifyResponse = try await api.post(
                Endpoints.usersMeMFATOTPVerify,
                body: VerifyBody(methodId: setup.methodId, code: totpCode))

            if let codes = response.recoveryCodes {
                recoveryCodes = codes
                totpSetupStep = .recovery
            } else {
                showTOTPSetup = false
                resetTOTPSetup()
                await loadMethods()
            }
        } catch {
            errorMessage = "Invalid code. Please try again."
        }

        isTOTPSetupInProgress = false
    }

    private func removeMethod(_ method: MFAMethod) async {
        isRemoving = true
        errorMessage = nil

        do {
            try await api.delete(Endpoints.usersMeMFAMethod(method.id))
            await loadMethods()
        } catch {
            errorMessage = "Failed to remove method: \(error.localizedDescription)"
        }

        isRemoving = false
    }

    private func regenerateRecoveryCodes() async {
        isRegenerating = true

        do {
            let response: RecoveryCodesResponse = try await api.post(Endpoints.usersMeMFARecoveryCodes)
            regeneratedCodes = response.codes
        } catch {
            errorMessage = "Failed to regenerate codes: \(error.localizedDescription)"
        }

        isRegenerating = false
    }

    private func resetTOTPSetup() {
        totpSetupStep = .name
        totpSetup = nil
        totpCode = ""
        totpName = "Authenticator App"
        recoveryCodes = nil
        isTOTPSetupInProgress = false
        errorMessage = nil
    }

    // MARK: - Helpers

    private func formatDate(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        if let date = formatter.date(from: iso) {
            return date.formatted(date: .abbreviated, time: .omitted)
        }
        return iso
    }
}

// MARK: - QR Code View

/// Generates a QR code from a string using CoreImage.
struct QRCodeView: View {
    let uri: String

    var body: some View {
        if let image = generateQRCode(from: uri) {
            Image(uiImage: image)
                .interpolation(.none)
                .resizable()
                .scaledToFit()
                .accessibilityLabel("QR code for authenticator app setup")
                .accessibilityHint("Scan this code with your authenticator app")
        } else {
            RoundedRectangle(cornerRadius: 8)
                .fill(Color(.secondarySystemGroupedBackground))
                .overlay(Text("QR Code").foregroundStyle(.secondary))
        }
    }

    private func generateQRCode(from string: String) -> UIImage? {
        let data = string.data(using: .ascii)
        guard let filter = CIFilter(name: "CIQRCodeGenerator") else { return nil }
        filter.setValue(data, forKey: "inputMessage")
        filter.setValue("M", forKey: "inputCorrectionLevel")

        guard let outputImage = filter.outputImage else { return nil }

        let scale = 200.0 / outputImage.extent.width
        let transformed = outputImage.transformed(by: CGAffineTransform(scaleX: scale, y: scale))

        let context = CIContext()
        guard let cgImage = context.createCGImage(transformed, from: transformed.extent) else {
            return nil
        }
        return UIImage(cgImage: cgImage)
    }
}

// MARK: - Recovery Codes View

/// Displays recovery codes with copy + save acknowledgment.
///
/// Mirrors the web client's `RecoveryCodesDisplay` component.
struct RecoveryCodesView: View {
    let codes: [String]
    let onDone: () -> Void

    @State private var hasSaved = false

    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: "shield.checkered")
                .font(.largeTitle)
                .foregroundStyle(.orange)

            Text("Save Your Recovery Codes")
                .font(.headline)

            Text("Store these codes in a safe place. Each code can only be used once. If you lose access to your authenticator, you can use these codes to sign in.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            // Codes grid
            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 8) {
                ForEach(codes, id: \.self) { code in
                    Text(code)
                        .font(.system(.caption, design: .monospaced))
                        .padding(.vertical, 6)
                        .frame(maxWidth: .infinity)
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 6))
                }
            }
            .padding(.horizontal)

            // Copy button
            Button {
                UIPasteboard.general.string = codes.joined(separator: "\n")
            } label: {
                Label("Copy All", systemImage: "doc.on.doc")
            }
            .buttonStyle(.bordered)

            // Save acknowledgment
            Toggle(isOn: $hasSaved) {
                Text("I have saved these recovery codes")
                    .font(.subheadline)
            }
            .padding(.horizontal)

            Button("Done") { onDone() }
                .buttonStyle(.borderedProminent)
                .disabled(!hasSaved)
                .padding(.horizontal)

            Spacer()
        }
        .padding(.top, 16)
    }
}
