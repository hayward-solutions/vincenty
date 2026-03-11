import SwiftUI

/// In-app diagnostic log viewer.
///
/// Displays the in-memory AppLogger ring buffer with:
/// - Level filter (debug / info / warn / error minimum)
/// - Per-category filter toggle
/// - Auto-scroll to newest entry
/// - Tap-to-expand rows that have detail text
/// - Clear All and Share (plain-text export) toolbar actions
struct SystemLogView: View {

    @State private var logger = AppLogger.shared
    @State private var minLevel: LogLevel = .debug
    @State private var selectedCategory: LogCategory? = nil
    @State private var expandedIds: Set<UUID> = []
    @State private var showShareSheet = false
    @State private var exportText = ""

    private var filtered: [LogEntry] {
        logger.entries.filter { entry in
            entry.level >= minLevel &&
            (selectedCategory == nil || entry.category == selectedCategory)
        }
    }

    var body: some View {
        VStack(spacing: 0) {
            filterBar
            Divider()
            logList
        }
        .navigationTitle("System Log")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button(role: .destructive) {
                    logger.clear()
                    expandedIds.removeAll()
                } label: {
                    Label("Clear", systemImage: "trash")
                }
                .disabled(logger.entries.isEmpty)
            }

            ToolbarItem(placement: .navigationBarTrailing) {
                Button {
                    exportText = logger.export()
                    showShareSheet = true
                } label: {
                    Label("Share", systemImage: "square.and.arrow.up")
                }
                .disabled(logger.entries.isEmpty)
            }
        }
        .sheet(isPresented: $showShareSheet) {
            ShareSheet(items: [exportText])
        }
    }

    // MARK: - Filter Bar

    private var filterBar: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                // Minimum level picker
                Menu {
                    ForEach(LogLevel.allCases, id: \.self) { level in
                        Button {
                            minLevel = level
                        } label: {
                            Label(level.label, systemImage: level.icon)
                        }
                    }
                } label: {
                    filterChip(
                        label: "≥ \(minLevel.label)",
                        isActive: true,
                        activeColor: minLevel.color
                    )
                }

                Divider().frame(height: 20)

                // Category filters
                Button {
                    selectedCategory = nil
                } label: {
                    filterChip(label: "All", isActive: selectedCategory == nil, activeColor: .accentColor)
                }

                ForEach(LogCategory.allCases, id: \.self) { cat in
                    Button {
                        selectedCategory = selectedCategory == cat ? nil : cat
                    } label: {
                        filterChip(
                            label: cat.rawValue,
                            isActive: selectedCategory == cat,
                            activeColor: .accentColor
                        )
                    }
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
        }
        .background(Color(.systemGroupedBackground))
    }

    @ViewBuilder
    private func filterChip(label: String, isActive: Bool, activeColor: Color) -> some View {
        Text(label)
            .font(.caption.weight(.medium))
            .foregroundStyle(isActive ? .white : .primary)
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(isActive ? activeColor : Color(.systemFill))
            .clipShape(Capsule())
    }

    // MARK: - Log List

    private var logList: some View {
        ScrollViewReader { proxy in
            List {
                if filtered.isEmpty {
                    ContentUnavailableView(
                        "No Entries",
                        systemImage: "list.bullet.rectangle",
                        description: Text(
                            logger.entries.isEmpty
                                ? "No log entries yet. Use the app and events will appear here."
                                : "No entries match the current filter."
                        )
                    )
                    .listRowBackground(Color.clear)
                } else {
                    ForEach(filtered) { entry in
                        LogEntryRow(
                            entry: entry,
                            isExpanded: expandedIds.contains(entry.id)
                        ) {
                            if expandedIds.contains(entry.id) {
                                expandedIds.remove(entry.id)
                            } else {
                                expandedIds.insert(entry.id)
                            }
                        }
                        .id(entry.id)
                    }
                }
            }
            .listStyle(.plain)
            // Auto-scroll to newest entry when list grows
            .onChange(of: logger.entries.count) { _, _ in
                if let last = filtered.last {
                    withAnimation(.easeOut(duration: 0.2)) {
                        proxy.scrollTo(last.id, anchor: .bottom)
                    }
                }
            }
        }
    }
}

// MARK: - Log Entry Row

private struct LogEntryRow: View {

    let entry: LogEntry
    let isExpanded: Bool
    let onTap: () -> Void

    private static let timeFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "HH:mm:ss.SSS"
        return f
    }()

    var body: some View {
        Button(action: onTap) {
            VStack(alignment: .leading, spacing: 3) {
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    // Timestamp (monospaced, fixed width)
                    Text(Self.timeFormatter.string(from: entry.timestamp))
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundStyle(.secondary)
                        .frame(width: 80, alignment: .leading)

                    // Level badge
                    Text(entry.level.label)
                        .font(.system(size: 9, weight: .bold))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 4)
                        .padding(.vertical, 1)
                        .background(entry.level.color)
                        .clipShape(RoundedRectangle(cornerRadius: 3))

                    // Category tag
                    Text(entry.category.rawValue)
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundStyle(entry.level.color)
                        .frame(width: 52, alignment: .leading)

                    // Message
                    Text(entry.message)
                        .font(.caption)
                        .foregroundStyle(.primary)
                        .lineLimit(isExpanded ? nil : 2)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }

                // Expanded detail (monospaced, indented)
                if isExpanded, let detail = entry.detail {
                    Text(detail)
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundStyle(.secondary)
                        .padding(.leading, 86)
                        .padding(.top, 2)
                        .textSelection(.enabled)
                }
            }
            .padding(.vertical, 3)
        }
        .buttonStyle(.plain)
        .listRowBackground(rowBackground)
    }

    private var rowBackground: Color? {
        switch entry.level {
        case .error:   return Color.red.opacity(0.07)
        case .warning: return Color.orange.opacity(0.06)
        default:       return nil
        }
    }
}

// MARK: - Share Sheet

private struct ShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ vc: UIActivityViewController, context: Context) {}
}
