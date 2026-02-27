import SwiftUI

/// Admin user management — list, create, edit, delete, reset MFA.
///
/// Mirrors the web client's `settings/server/users/page.tsx`:
/// - Paginated user list with role/MFA/status badges
/// - Create user sheet
/// - Edit user sheet (email, display name, password, admin, active)
/// - Reset MFA, delete user
struct AdminUsersView: View {
    @State private var users: [User] = []
    @State private var totalCount = 0
    @State private var currentPage = 1
    @State private var isLoading = true
    @State private var errorMessage: String?

    // Create
    @State private var showCreateSheet = false
    @State private var createUsername = ""
    @State private var createEmail = ""
    @State private var createPassword = ""
    @State private var createDisplayName = ""
    @State private var createIsAdmin = false
    @State private var isCreating = false

    // Edit
    @State private var editUser: User?
    @State private var showEditSheet = false
    @State private var editEmail = ""
    @State private var editDisplayName = ""
    @State private var editPassword = ""
    @State private var editIsAdmin = false
    @State private var editIsActive = true
    @State private var isEditing = false

    // Delete / MFA reset
    @State private var userToDelete: User?
    @State private var showDeleteAlert = false
    @State private var userToResetMFA: User?
    @State private var showResetMFAAlert = false

    private let pageSize = 20
    private let api = APIClient.shared

    var body: some View {
        List {
            if isLoading && users.isEmpty {
                Section { ProgressView().frame(maxWidth: .infinity) }
            } else if users.isEmpty {
                Section { Text("No users found").foregroundStyle(.secondary) }
            } else {
                Section {
                    ForEach(users) { user in
                        userRow(user)
                    }
                }

                // Pagination
                if totalCount > pageSize {
                    Section {
                        HStack {
                            Button("Previous") {
                                currentPage = max(1, currentPage - 1)
                                Task { await loadUsers() }
                            }
                            .disabled(currentPage <= 1)

                            Spacer()

                            let start = (currentPage - 1) * pageSize + 1
                            let end = min(currentPage * pageSize, totalCount)
                            Text("\(start)-\(end) of \(totalCount)")
                                .font(.caption)
                                .foregroundStyle(.secondary)

                            Spacer()

                            Button("Next") {
                                currentPage += 1
                                Task { await loadUsers() }
                            }
                            .disabled(currentPage * pageSize >= totalCount)
                        }
                    }
                }
            }

            if let error = errorMessage {
                Section {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
        }
        .navigationTitle("Users")
        .task { await loadUsers() }
        .refreshable { await loadUsers() }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    resetCreateForm()
                    showCreateSheet = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(isPresented: $showCreateSheet) { createUserSheet }
        .sheet(isPresented: $showEditSheet) { editUserSheet }
        .alert("Delete User", isPresented: $showDeleteAlert) {
            Button("Cancel", role: .cancel) {}
            Button("Delete", role: .destructive) {
                if let user = userToDelete {
                    Task { await deleteUser(user) }
                }
            }
        } message: {
            Text("Permanently delete \"\(userToDelete?.username ?? "")\"? This cannot be undone.")
        }
        .alert("Reset MFA", isPresented: $showResetMFAAlert) {
            Button("Cancel", role: .cancel) {}
            Button("Reset", role: .destructive) {
                if let user = userToResetMFA {
                    Task { await resetMFA(user) }
                }
            }
        } message: {
            Text("Reset all MFA methods for \"\(userToResetMFA?.username ?? "")\"? They will need to set up MFA again.")
        }
    }

    // MARK: - User Row

    @ViewBuilder
    private func userRow(_ user: User) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(user.username)
                        .font(.subheadline.weight(.medium))
                    if !user.displayName.isEmpty {
                        Text(user.displayName)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Text(user.email)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Spacer()

                VStack(alignment: .trailing, spacing: 4) {
                    HStack(spacing: 4) {
                        if user.isAdmin {
                            statusBadge("Admin", color: .blue)
                        }
                        if user.mfaEnabled {
                            statusBadge("MFA", color: .green)
                        }
                        if !user.isActive {
                            statusBadge("Inactive", color: .red)
                        }
                    }
                }
            }

            // Action buttons
            HStack(spacing: 12) {
                Button {
                    editUser = user
                    editEmail = user.email
                    editDisplayName = user.displayName
                    editPassword = ""
                    editIsAdmin = user.isAdmin
                    editIsActive = user.isActive
                    showEditSheet = true
                } label: {
                    Label("Edit", systemImage: "pencil")
                        .font(.caption)
                }

                if user.mfaEnabled {
                    Button {
                        userToResetMFA = user
                        showResetMFAAlert = true
                    } label: {
                        Label("Reset MFA", systemImage: "key.slash")
                            .font(.caption)
                    }
                }

                Spacer()

                Button(role: .destructive) {
                    userToDelete = user
                    showDeleteAlert = true
                } label: {
                    Label("Delete", systemImage: "trash")
                        .font(.caption)
                }
            }
        }
        .padding(.vertical, 4)
    }

    private func statusBadge(_ text: String, color: Color) -> some View {
        Text(text)
            .font(.caption2.weight(.medium))
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(color.opacity(0.15))
            .foregroundStyle(color)
            .clipShape(RoundedRectangle(cornerRadius: 4))
    }

    // MARK: - Create User Sheet

    @ViewBuilder
    private var createUserSheet: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Username", text: $createUsername)
                        .textInputAutocapitalization(.never)
                        .textContentType(.username)
                    TextField("Email", text: $createEmail)
                        .keyboardType(.emailAddress)
                        .textInputAutocapitalization(.never)
                    SecureField("Password (min 8 chars)", text: $createPassword)
                        .textContentType(.newPassword)
                    TextField("Display Name (optional)", text: $createDisplayName)
                    Toggle("Administrator", isOn: $createIsAdmin)
                }
            }
            .navigationTitle("Create User")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showCreateSheet = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") {
                        Task { await createUser() }
                    }
                    .disabled(
                        createUsername.isEmpty || createEmail.isEmpty
                            || createPassword.count < 8 || isCreating)
                }
            }
        }
    }

    // MARK: - Edit User Sheet

    @ViewBuilder
    private var editUserSheet: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Email", text: $editEmail)
                        .keyboardType(.emailAddress)
                        .textInputAutocapitalization(.never)
                    TextField("Display Name", text: $editDisplayName)
                    SecureField("New Password (leave blank to keep)", text: $editPassword)
                    Toggle("Administrator", isOn: $editIsAdmin)
                    Toggle("Active", isOn: $editIsActive)
                }
            }
            .navigationTitle("Edit User")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showEditSheet = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await updateUser() }
                    }
                    .disabled(isEditing)
                }
            }
        }
    }

    // MARK: - API Actions

    private func loadUsers() async {
        isLoading = true
        errorMessage = nil

        do {
            let params: [String: String] = [
                "page": String(currentPage),
                "page_size": String(pageSize),
            ]
            let response: ListResponse<User> = try await api.get(Endpoints.users, params: params)
            users = response.data
            totalCount = response.total
        } catch {
            errorMessage = "Failed to load users"
        }

        isLoading = false
    }

    private func createUser() async {
        isCreating = true
        errorMessage = nil

        do {
            let body = CreateUserRequest(
                username: createUsername,
                email: createEmail,
                password: createPassword,
                displayName: createDisplayName.isEmpty ? nil : createDisplayName,
                isAdmin: createIsAdmin ? true : nil)
            let _: User = try await api.post(Endpoints.users, body: body)
            showCreateSheet = false
            await loadUsers()
        } catch {
            errorMessage = "Failed to create user"
        }

        isCreating = false
    }

    private func updateUser() async {
        guard let user = editUser else { return }
        isEditing = true
        errorMessage = nil

        do {
            let body = UpdateUserRequest(
                email: editEmail,
                displayName: editDisplayName,
                password: editPassword.isEmpty ? nil : editPassword,
                isAdmin: editIsAdmin,
                isActive: editIsActive)
            let _: User = try await api.put(Endpoints.user(user.id), body: body)
            showEditSheet = false
            await loadUsers()
        } catch {
            errorMessage = "Failed to update user"
        }

        isEditing = false
    }

    private func deleteUser(_ user: User) async {
        do {
            try await api.delete(Endpoints.user(user.id))
            await loadUsers()
        } catch {
            errorMessage = "Failed to delete user"
        }
    }

    private func resetMFA(_ user: User) async {
        do {
            try await api.delete(Endpoints.userMFA(user.id))
            await loadUsers()
        } catch {
            errorMessage = "Failed to reset MFA"
        }
    }

    private func resetCreateForm() {
        createUsername = ""
        createEmail = ""
        createPassword = ""
        createDisplayName = ""
        createIsAdmin = false
    }
}
