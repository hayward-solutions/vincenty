import SwiftUI

/// Sheet for starting a new direct message by selecting a user.
/// Mirrors the web client's `new-dm-dialog.tsx`.
struct NewDirectMessageView: View {
    let onSelect: (String, String) -> Void  // (userId, displayName)
    @Environment(\.dismiss) private var dismiss

    @State private var users: [User] = []
    @State private var isLoading = false
    @State private var search = ""

    private let api = APIClient.shared

    var body: some View {
        NavigationStack {
            contentView
                .searchable(text: $search, prompt: "Search users...")
                .navigationTitle("New Message")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { dismiss() }
                    }
                }
                .task {
                    await loadUsers()
                }
        }
    }
    
    @ViewBuilder
    private var contentView: some View {
        if isLoading {
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if filteredUsers.isEmpty {
            ContentUnavailableView.search(text: search)
        } else {
            List(filteredUsers) { user in
                Button {
                    let name = user.displayName.isEmpty ? user.username : user.displayName
                    onSelect(user.id, name)
                    dismiss()
                } label: {
                    HStack(spacing: 10) {
                        Image(systemName: "person.circle")
                            .font(.title3)
                            .foregroundStyle(.secondary)

                        VStack(alignment: .leading, spacing: 2) {
                            Text(user.displayName.isEmpty ? user.username : user.displayName)
                                .font(.subheadline.weight(.medium))
                            if !user.displayName.isEmpty {
                                Text("@\(user.username)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
            }
        }
    }

    // MARK: - Data

    private var filteredUsers: [User] {
        if search.isEmpty { return users }
        return users.filter {
            $0.username.localizedCaseInsensitiveContains(search)
                || $0.displayName.localizedCaseInsensitiveContains(search)
        }
    }

    private func loadUsers() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let response: ListResponse<User> = try await api.get(
                Endpoints.users, params: ["page_size": "200"])
            users = response.data
        } catch {
            // Silently fail
        }
    }
}
