import SwiftUI

/// Admin audit logs — global audit log viewer with filters and export.
///
/// Mirrors the web client's `settings/server/audit-logs/page.tsx`:
/// - Paginated table of all audit events across all users
/// - Filter by action, resource type, date range
/// - Export to CSV/JSON
struct AdminAuditLogsView: View {
    @State private var logs: [AuditLogResponse] = []
    @State private var totalCount = 0
    @State private var currentPage = 1
    @State private var isLoading = true
    @State private var errorMessage: String?

    // Filters
    @State private var selectedAction: String?
    @State private var selectedResourceType: String?
    @State private var showFilters = false

    private let pageSize = 20
    private let api = APIClient.shared

    var body: some View {
        List {
            // Filter toggle
            Section {
                Button {
                    showFilters.toggle()
                } label: {
                    Label(
                        showFilters ? "Hide Filters" : "Show Filters",
                        systemImage: "line.3.horizontal.decrease.circle")
                }
            }

            // Filters
            if showFilters {
                Section("Filters") {
                    Picker("Action", selection: $selectedAction) {
                        Text("All Actions").tag(nil as String?)
                        ForEach(AuditFilters.knownActions, id: \.self) { action in
                            Text(action).tag(action as String?)
                        }
                    }

                    Picker("Resource Type", selection: $selectedResourceType) {
                        Text("All Types").tag(nil as String?)
                        ForEach(AuditFilters.knownResourceTypes, id: \.self) { type in
                            Text(type).tag(type as String?)
                        }
                    }

                    HStack {
                        Button("Apply") {
                            currentPage = 1
                            Task { await loadLogs() }
                        }
                        .font(.subheadline)

                        Spacer()

                        Button("Reset") {
                            selectedAction = nil
                            selectedResourceType = nil
                            currentPage = 1
                            Task { await loadLogs() }
                        }
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    }
                }
            }

            // Log entries
            if isLoading && logs.isEmpty {
                Section {
                    ForEach(0..<5, id: \.self) { _ in
                        RoundedRectangle(cornerRadius: 4)
                            .fill(Color(.systemGray5))
                            .frame(height: 40)
                    }
                }
            } else if logs.isEmpty {
                Section {
                    Text("No audit logs found")
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("Audit Logs") {
                    ForEach(logs) { log in
                        logRow(log)
                    }
                }

                // Pagination
                if totalCount > pageSize {
                    Section {
                        HStack {
                            Button("Previous") {
                                currentPage = max(1, currentPage - 1)
                                Task { await loadLogs() }
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
                                Task { await loadLogs() }
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
        .navigationTitle("Audit Logs")
        .task { await loadLogs() }
        .refreshable { await loadLogs() }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Menu {
                    Button {
                        Task { await exportLogs(format: "csv") }
                    } label: {
                        Label("Export CSV", systemImage: "tablecells")
                    }
                    Button {
                        Task { await exportLogs(format: "json") }
                    } label: {
                        Label("Export JSON", systemImage: "curlybraces")
                    }
                } label: {
                    Image(systemName: "square.and.arrow.up")
                }
            }
        }
    }

    // MARK: - Log Row

    @ViewBuilder
    private func logRow(_ log: AuditLogResponse) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 6) {
                Text(formatDate(log.createdAt))
                    .font(.caption2)
                    .foregroundStyle(.secondary)

                Spacer()

                Text(log.displayName.isEmpty ? log.username : log.displayName)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 6) {
                // Action badge with color coding
                Text(log.action)
                    .font(.caption2.weight(.medium))
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(actionColor(log.action).opacity(0.15))
                    .foregroundStyle(actionColor(log.action))
                    .clipShape(RoundedRectangle(cornerRadius: 4))

                if !log.resourceType.isEmpty {
                    Text(log.resourceType)
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                }

                Spacer()

                Text(log.ipAddress)
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.vertical, 2)
    }

    // MARK: - API Actions

    private func loadLogs() async {
        isLoading = true
        errorMessage = nil

        var filters = AuditFilters()
        filters.action = selectedAction
        filters.resourceType = selectedResourceType
        filters.page = currentPage
        filters.pageSize = pageSize

        do {
            let response: ListResponse<AuditLogResponse> = try await api.get(
                Endpoints.auditLogs,
                params: filters.queryParams)
            logs = response.data
            totalCount = response.total ?? 0
        } catch {
            errorMessage = "Failed to load audit logs"
        }

        isLoading = false
    }

    private func exportLogs(format: String) async {
        var params: [String: String] = ["format": format]
        if let action = selectedAction { params["action"] = action }
        if let resourceType = selectedResourceType { params["resource_type"] = resourceType }

        do {
            let data = try await api.download(Endpoints.auditLogsExport, params: params)

            let ext = format == "csv" ? "csv" : "json"
            let tempURL = FileManager.default.temporaryDirectory
                .appendingPathComponent("audit-logs-export.\(ext)")
            try data.write(to: tempURL)

            await MainActor.run {
                guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
                      let rootVC = windowScene.windows.first?.rootViewController
                else { return }
                let activityVC = UIActivityViewController(
                    activityItems: [tempURL], applicationActivities: nil)
                rootVC.present(activityVC, animated: true)
            }
        } catch {
            errorMessage = "Export failed: \(error.localizedDescription)"
        }
    }

    // MARK: - Helpers

    private func actionColor(_ action: String) -> Color {
        if action.contains("create") || action.contains("add") { return .green }
        if action.contains("delete") || action.contains("remove") { return .red }
        if action.hasPrefix("auth.") { return .blue }
        return .secondary
    }

    private func formatDate(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        guard let date = formatter.date(from: iso) else { return iso }
        return date.formatted(date: .abbreviated, time: .shortened)
    }
}
