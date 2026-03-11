import PhotosUI
import SwiftUI

/// Message input bar with text field and attachment support.
/// Mirrors the web client's `message-input.tsx`.
struct MessageInputView: View {
    let onSend: (String, [URL]) -> Void
    let disabled: Bool

    @State private var text = ""
    @State private var selectedPhotos: [PhotosPickerItem] = []
    @State private var attachedFiles: [AttachedFile] = []
    @State private var showFilePicker = false

    private static let maxFileSize = 25 * 1024 * 1024  // 25 MB

    var body: some View {
        VStack(spacing: 0) {
            // File previews
            if !attachedFiles.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(attachedFiles) { file in
                            HStack(spacing: 4) {
                                Text(file.name)
                                    .font(.caption)
                                    .lineLimit(1)
                                    .frame(maxWidth: 120)

                                Button {
                                    removeFile(file)
                                } label: {
                                    Image(systemName: "xmark.circle.fill")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(.secondary.opacity(0.15))
                            .clipShape(RoundedRectangle(cornerRadius: 6))
                        }
                    }
                    .padding(.horizontal)
                    .padding(.vertical, 6)
                }
            }

            Divider()

            // Input row
            HStack(spacing: 8) {
                // Attachment button
                Menu {
                    Button {
                        showFilePicker = true
                    } label: {
                        Label("Choose File", systemImage: "doc")
                    }

                    PhotosPicker(
                        selection: $selectedPhotos,
                        maxSelectionCount: 5,
                        matching: .any(of: [.images, .videos])
                    ) {
                        Label("Photo Library", systemImage: "photo")
                    }
                } label: {
                    Image(systemName: "paperclip")
                        .font(.system(size: 18))
                        .foregroundStyle(.secondary)
                }
                .disabled(disabled)

                // Text field
                TextField("Type a message...", text: $text)
                    .textFieldStyle(.plain)
                    .font(.subheadline)
                    .disabled(disabled)
                    .onSubmit {
                        handleSend()
                    }

                // Send button
                Button {
                    handleSend()
                } label: {
                    Image(systemName: "arrow.up.circle.fill")
                        .font(.system(size: 28))
                        .foregroundStyle(canSend ? .blue : .secondary.opacity(0.5))
                }
                .disabled(!canSend || disabled)
            }
            .padding(.horizontal)
            .padding(.vertical, 8)
        }
        .background(.bar)
        .fileImporter(
            isPresented: $showFilePicker,
            allowedContentTypes: [.data],
            allowsMultipleSelection: true
        ) { result in
            if case .success(let urls) = result {
                for url in urls {
                    addFile(url: url, name: url.lastPathComponent)
                }
            }
        }
        .onChange(of: selectedPhotos) { _, items in
            Task {
                for item in items {
                    if let data = try? await item.loadTransferable(type: Data.self) {
                        let tempURL = FileManager.default.temporaryDirectory
                            .appendingPathComponent(UUID().uuidString + ".jpg")
                        try? data.write(to: tempURL)
                        addFile(url: tempURL, name: "Photo.jpg")
                    }
                }
                selectedPhotos = []
            }
        }
    }

    // MARK: - Computed

    private var canSend: Bool {
        !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || !attachedFiles.isEmpty
    }

    // MARK: - Actions

    private func handleSend() {
        let content = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty || !attachedFiles.isEmpty else { return }

        let urls = attachedFiles.map(\.url)
        onSend(content, urls)
        text = ""
        attachedFiles = []
    }

    private func addFile(url: URL, name: String) {
        // Check size
        if let attrs = try? FileManager.default.attributesOfItem(atPath: url.path),
           let size = attrs[.size] as? Int, size > Self.maxFileSize
        {
            // Skip oversized files
            return
        }
        attachedFiles.append(AttachedFile(url: url, name: name))
    }

    private func removeFile(_ file: AttachedFile) {
        attachedFiles.removeAll { $0.id == file.id }
    }
}

// MARK: - Attached File

private struct AttachedFile: Identifiable {
    let id = UUID()
    let url: URL
    let name: String
}
