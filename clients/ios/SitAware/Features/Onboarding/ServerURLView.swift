import SwiftUI

/// First-launch screen to configure the server URL.
/// iOS-specific — the web client is co-deployed with the API and doesn't need this.
struct ServerURLView: View {
    @Environment(AuthManager.self) private var auth
    @State private var serverURL = ""
    @State private var isValidating = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            VStack(spacing: 32) {
                Spacer()

                // Logo area
                VStack(spacing: 12) {
                    Image(systemName: "antenna.radiowaves.left.and.right")
                        .font(.system(size: 56))
                        .foregroundStyle(.tint)

                    Text("SitAware")
                        .font(.largeTitle.bold())

                    Text("Situational Awareness Platform")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                // Server URL input
                VStack(alignment: .leading, spacing: 8) {
                    Text("Server URL")
                        .font(.headline)

                    TextField("https://sa.example.com", text: $serverURL)
                        .textFieldStyle(.roundedBorder)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .keyboardType(.URL)
                        .submitLabel(.go)
                        .onSubmit { Task { await validate() } }

                    Text("Enter the URL of your SitAware server.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .padding(.horizontal)

                if let error {
                    Text(error)
                        .font(.callout)
                        .foregroundStyle(.red)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal)
                }

                Button(action: { Task { await validate() } }) {
                    if isValidating {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                    } else {
                        Text("Connect")
                            .frame(maxWidth: .infinity)
                    }
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.large)
                .disabled(serverURL.trimmingCharacters(in: .whitespaces).isEmpty || isValidating)
                .padding(.horizontal)

                Spacer()
                Spacer()
            }
        }
    }

    private func validate() async {
        let trimmed = serverURL.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else { return }

        // Normalize: ensure https prefix, strip trailing slash
        var url = trimmed
        if !url.hasPrefix("http://") && !url.hasPrefix("https://") {
            url = "https://\(url)"
        }
        url = url.trimmingSuffix(while: { $0 == "/" })

        isValidating = true
        error = nil

        do {
            // Hit the health endpoint to verify the server is reachable
            guard let healthURL = URL(string: "\(url)/healthz") else {
                error = "Invalid URL format."
                isValidating = false
                return
            }

            let (_, response) = try await URLSession.shared.data(from: healthURL)

            guard let httpResponse = response as? HTTPURLResponse,
                  httpResponse.statusCode == 200
            else {
                error = "Server responded but health check failed. Is this a SitAware server?"
                isValidating = false
                return
            }

            // Success — save the server URL and trigger navigation
            KeychainStore.shared.serverURL = url
            auth.hasServerURL = true
        } catch {
            self.error = "Could not connect to server. Check the URL and try again."
        }

        isValidating = false
    }
}

private extension String {
    func trimmingSuffix(while predicate: (Character) -> Bool) -> String {
        var result = self
        while let last = result.last, predicate(last) {
            result.removeLast()
        }
        return result
    }
}

