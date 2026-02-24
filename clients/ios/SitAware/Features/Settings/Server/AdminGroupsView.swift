import SwiftUI

/// Admin group management — list, create, edit, delete groups + member management.
///
/// Mirrors the web client's `settings/server/groups/page.tsx` + group detail page:
/// - Paginated group list with member counts
/// - Create/edit/delete groups
/// - Group detail: marker editor, member management (add/edit permissions/remove), audit log
struct AdminGroupsView: View {
    @State private var groups: [Group] = []
    @State private var totalCount = 0
    @State private var currentPage = 1
    @State private var isLoading = true
    @State private var errorMessage: String?

    // Create
    @State private var showCreateSheet = false
    @State private var createName = ""
    @State private var createDescription = ""
    @State private var isCreating = false

    // Edit
    @State private var editGroup: Group?
    @State private var showEditSheet = false
    @State private var editName = ""
    @State private var editDescription = ""
    @State private var isEditing = false

    // Delete
    @State private var groupToDelete: Group?
    @State private var showDeleteAlert = false

    // Detail navigation
    @State private var selectedGroup: Group?
    @State private var showDetail = false

    private let pageSize = 20
    private let api = APIClient.shared

    var body: some View {
        List {
            if isLoading && groups.isEmpty {
                Section { ProgressView().frame(maxWidth: .infinity) }
            } else if groups.isEmpty {
                Section { Text("No groups found").foregroundStyle(.secondary) }
            } else {
                Section {
                    ForEach(groups) { group in
                        groupRow(group)
                    }
                }

                // Pagination
                if totalCount > pageSize {
                    Section {
                        HStack {
                            Button("Previous") {
                                currentPage = max(1, currentPage - 1)
                                Task { await loadGroups() }
                            }
                            .disabled(currentPage <= 1)
                            Spacer()
                            let start = (currentPage - 1) * pageSize + 1
                            let end = min(currentPage * pageSize, totalCount)
                            Text("\(start)-\(end) of \(totalCount)")
                                .font(.caption).foregroundStyle(.secondary)
                            Spacer()
                            Button("Next") {
                                currentPage += 1
                                Task { await loadGroups() }
                            }
                            .disabled(currentPage * pageSize >= totalCount)
                        }
                    }
                }
            }

            if let error = errorMessage {
                Section { Text(error).foregroundStyle(.red).font(.caption) }
            }
        }
        .navigationTitle("Groups")
        .task { await loadGroups() }
        .refreshable { await loadGroups() }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    createName = ""
                    createDescription = ""
                    showCreateSheet = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .sheet(isPresented: $showCreateSheet) { createGroupSheet }
        .sheet(isPresented: $showEditSheet) { editGroupSheet }
        .navigationDestination(isPresented: $showDetail) {
            if let group = selectedGroup {
                AdminGroupDetailView(groupId: group.id, groupName: group.name)
            }
        }
        .alert("Delete Group", isPresented: $showDeleteAlert) {
            Button("Cancel", role: .cancel) {}
            Button("Delete", role: .destructive) {
                if let group = groupToDelete {
                    Task { await deleteGroup(group) }
                }
            }
        } message: {
            Text("Delete \"\(groupToDelete?.name ?? "")\"? This will remove all members.")
        }
    }

    // MARK: - Group Row

    @ViewBuilder
    private func groupRow(_ group: Group) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 10) {
                Circle()
                    .fill(Color(hex: group.markerColor.isEmpty ? "#3b82f6" : group.markerColor) ?? .blue)
                    .frame(width: 28, height: 28)
                    .overlay(
                        Text(String(group.name.prefix(1)).uppercased())
                            .font(.caption2.weight(.bold))
                            .foregroundStyle(.white))

                VStack(alignment: .leading, spacing: 2) {
                    Text(group.name)
                        .font(.subheadline.weight(.medium))
                    if !group.description.isEmpty {
                        Text(group.description)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(1)
                    }
                }

                Spacer()

                Text("\(group.memberCount) members")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 12) {
                Button {
                    selectedGroup = group
                    showDetail = true
                } label: {
                    Label("Members", systemImage: "person.3")
                        .font(.caption)
                }

                Button {
                    editGroup = group
                    editName = group.name
                    editDescription = group.description
                    showEditSheet = true
                } label: {
                    Label("Edit", systemImage: "pencil")
                        .font(.caption)
                }

                Spacer()

                Button(role: .destructive) {
                    groupToDelete = group
                    showDeleteAlert = true
                } label: {
                    Label("Delete", systemImage: "trash")
                        .font(.caption)
                }
            }
        }
        .padding(.vertical, 4)
    }

    // MARK: - Create Sheet

    @ViewBuilder
    private var createGroupSheet: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Group Name", text: $createName)
                    TextField("Description (optional)", text: $createDescription)
                }
            }
            .navigationTitle("Create Group")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showCreateSheet = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") {
                        Task { await createGroup() }
                    }
                    .disabled(createName.trimmingCharacters(in: .whitespaces).isEmpty || isCreating)
                }
            }
        }
    }

    // MARK: - Edit Sheet

    @ViewBuilder
    private var editGroupSheet: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Group Name", text: $editName)
                    TextField("Description", text: $editDescription)
                }
            }
            .navigationTitle("Edit Group")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showEditSheet = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await updateGroup() }
                    }
                    .disabled(editName.trimmingCharacters(in: .whitespaces).isEmpty || isEditing)
                }
            }
        }
    }

    // MARK: - API Actions

    private func loadGroups() async {
        isLoading = true
        errorMessage = nil

        do {
            let params: [String: String] = [
                "page": String(currentPage),
                "page_size": String(pageSize),
            ]
            let response: ListResponse<Group> = try await api.get(Endpoints.groups, params: params)
            groups = response.data
            totalCount = response.total ?? 0
        } catch {
            errorMessage = "Failed to load groups"
        }

        isLoading = false
    }

    private func createGroup() async {
        isCreating = true
        do {
            let body = CreateGroupRequest(
                name: createName,
                description: createDescription.isEmpty ? nil : createDescription)
            let _: Group = try await api.post(Endpoints.groups, body: body)
            showCreateSheet = false
            await loadGroups()
        } catch {
            errorMessage = "Failed to create group"
        }
        isCreating = false
    }

    private func updateGroup() async {
        guard let group = editGroup else { return }
        isEditing = true
        do {
            let body = UpdateGroupRequest(
                name: editName,
                description: editDescription.isEmpty ? nil : editDescription)
            let _: Group = try await api.put(Endpoints.group(group.id), body: body)
            showEditSheet = false
            await loadGroups()
        } catch {
            errorMessage = "Failed to update group"
        }
        isEditing = false
    }

    private func deleteGroup(_ group: Group) async {
        do {
            try await api.delete(Endpoints.group(group.id))
            await loadGroups()
        } catch {
            errorMessage = "Failed to delete group"
        }
    }
}

// MARK: - Group Detail View

/// Admin group detail — marker editor, member management.
struct AdminGroupDetailView: View {
    let groupId: String
    let groupName: String

    @State private var members: [GroupMember] = []
    @State private var isLoading = true
    @State private var errorMessage: String?

    // Add member
    @State private var showAddMember = false
    @State private var availableUsers: [User] = []
    @State private var selectedUserId: String?
    @State private var addCanRead = true
    @State private var addCanWrite = false
    @State private var addIsAdmin = false
    @State private var isAdding = false

    // Edit permissions
    @State private var editMember: GroupMember?
    @State private var showEditPermissions = false
    @State private var permCanRead = true
    @State private var permCanWrite = false
    @State private var permIsAdmin = false

    // Remove
    @State private var memberToRemove: GroupMember?
    @State private var showRemoveAlert = false

    private let api = APIClient.shared

    var body: some View {
        List {
            // Members
            if isLoading {
                Section { ProgressView().frame(maxWidth: .infinity) }
            } else {
                Section("Members (\(members.count))") {
                    ForEach(members) { member in
                        memberRow(member)
                    }
                }
            }

            if let error = errorMessage {
                Section { Text(error).foregroundStyle(.red).font(.caption) }
            }
        }
        .navigationTitle(groupName)
        .task { await loadMembers() }
        .refreshable { await loadMembers() }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    Task { await prepareAddMember() }
                } label: {
                    Image(systemName: "person.badge.plus")
                }
            }
        }
        .sheet(isPresented: $showAddMember) { addMemberSheet }
        .sheet(isPresented: $showEditPermissions) { editPermissionsSheet }
        .alert("Remove Member", isPresented: $showRemoveAlert) {
            Button("Cancel", role: .cancel) {}
            Button("Remove", role: .destructive) {
                if let member = memberToRemove {
                    Task { await removeMember(member) }
                }
            }
        } message: {
            Text("Remove \"\(memberToRemove?.username ?? "")\" from this group?")
        }
    }

    // MARK: - Member Row

    @ViewBuilder
    private func memberRow(_ member: GroupMember) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(member.username)
                        .font(.subheadline.weight(.medium))
                    if !member.displayName.isEmpty {
                        Text(member.displayName)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                Spacer()

                HStack(spacing: 4) {
                    if member.canRead { BadgeView(text: "Read", color: .blue) }
                    if member.canWrite { BadgeView(text: "Write", color: .green) }
                    if member.isGroupAdmin { BadgeView(text: "Admin", color: .orange) }
                }
            }

            HStack(spacing: 12) {
                Button {
                    editMember = member
                    permCanRead = member.canRead
                    permCanWrite = member.canWrite
                    permIsAdmin = member.isGroupAdmin
                    showEditPermissions = true
                } label: {
                    Label("Permissions", systemImage: "lock.shield")
                        .font(.caption)
                }

                Spacer()

                Button(role: .destructive) {
                    memberToRemove = member
                    showRemoveAlert = true
                } label: {
                    Label("Remove", systemImage: "person.badge.minus")
                        .font(.caption)
                }
            }
        }
        .padding(.vertical, 4)
    }

    // Permission badges now use shared BadgeView component

    // MARK: - Add Member Sheet

    @ViewBuilder
    private var addMemberSheet: some View {
        NavigationStack {
            Form {
                Section("Select User") {
                    Picker("User", selection: $selectedUserId) {
                        Text("Select a user").tag(nil as String?)
                        ForEach(availableUsers) { user in
                            Text("\(user.username) (\(user.displayName))")
                                .tag(user.id as String?)
                        }
                    }
                }

                Section("Permissions") {
                    Toggle("Can Read", isOn: $addCanRead)
                    Toggle("Can Write", isOn: $addCanWrite)
                    Toggle("Group Admin", isOn: $addIsAdmin)
                }
            }
            .navigationTitle("Add Member")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showAddMember = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Add") {
                        Task { await addMember() }
                    }
                    .disabled(selectedUserId == nil || isAdding)
                }
            }
        }
    }

    // MARK: - Edit Permissions Sheet

    @ViewBuilder
    private var editPermissionsSheet: some View {
        NavigationStack {
            Form {
                Section("Permissions for \(editMember?.username ?? "")") {
                    Toggle("Can Read", isOn: $permCanRead)
                    Toggle("Can Write", isOn: $permCanWrite)
                    Toggle("Group Admin", isOn: $permIsAdmin)
                }
            }
            .navigationTitle("Edit Permissions")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showEditPermissions = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await updatePermissions() }
                    }
                }
            }
        }
    }

    // MARK: - API Actions

    private func loadMembers() async {
        isLoading = true
        do {
            let response: ListResponse<GroupMember> = try await api.get(
                Endpoints.groupMembers(groupId))
            members = response.data
        } catch {
            errorMessage = "Failed to load members"
        }
        isLoading = false
    }

    private func prepareAddMember() async {
        do {
            let response: ListResponse<User> = try await api.get(
                Endpoints.users, params: ["page_size": "200"])
            let memberUserIds = Set(members.map(\.userId))
            availableUsers = response.data.filter { $0.isActive && !memberUserIds.contains($0.id) }
            selectedUserId = nil
            addCanRead = true
            addCanWrite = false
            addIsAdmin = false
            showAddMember = true
        } catch {
            errorMessage = "Failed to load users"
        }
    }

    private func addMember() async {
        guard let userId = selectedUserId else { return }
        isAdding = true
        do {
            let body = AddGroupMemberRequest(
                userId: userId, canRead: addCanRead, canWrite: addCanWrite, isGroupAdmin: addIsAdmin)
            let _: GroupMember = try await api.post(Endpoints.groupMembers(groupId), body: body)
            showAddMember = false
            await loadMembers()
        } catch {
            errorMessage = "Failed to add member"
        }
        isAdding = false
    }

    private func updatePermissions() async {
        guard let member = editMember else { return }
        do {
            let body = UpdateGroupMemberRequest(
                canRead: permCanRead, canWrite: permCanWrite, isGroupAdmin: permIsAdmin)
            let _: GroupMember = try await api.put(
                Endpoints.groupMember(groupId, member.userId), body: body)
            showEditPermissions = false
            await loadMembers()
        } catch {
            errorMessage = "Failed to update permissions"
        }
    }

    private func removeMember(_ member: GroupMember) async {
        do {
            try await api.delete(Endpoints.groupMember(groupId, member.userId))
            await loadMembers()
        } catch {
            errorMessage = "Failed to remove member"
        }
    }
}
