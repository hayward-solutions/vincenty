import SwiftUI

/// A single message bubble in the chat thread.
/// Mirrors the web client's `message-bubble.tsx`.
struct MessageBubbleView: View {
    let message: MessageResponse
    let isOwn: Bool

    var body: some View {
        VStack(alignment: isOwn ? .trailing : .leading, spacing: 4) {
            // Sender name (not for own messages)
            if !isOwn {
                Text(message.displayName.isEmpty ? message.username : message.displayName)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            // Bubble
            VStack(alignment: .leading, spacing: 0) {
                // Text content
                if !message.content.isEmpty {
                    Text(message.content)
                        .font(.subheadline)
                        .padding(.horizontal, 12)
                        .padding(.top, 8)
                        .padding(.bottom, message.attachments.isEmpty ? 8 : 4)
                }

                // Attachments
                if !message.attachments.isEmpty {
                    ForEach(message.attachments) { attachment in
                        if attachment.isImage {
                            // Image attachment — show thumbnail
                            AsyncImage(url: attachmentURL(for: attachment)) { phase in
                                switch phase {
                                case .success(let image):
                                    image
                                        .resizable()
                                        .aspectRatio(contentMode: .fill)
                                        .frame(maxWidth: .infinity, maxHeight: 200)
                                        .clipped()
                                case .failure:
                                    fileAttachmentRow(attachment)
                                default:
                                    ProgressView()
                                        .frame(maxWidth: .infinity, minHeight: 80)
                                }
                            }
                        } else {
                            fileAttachmentRow(attachment)
                        }
                    }
                }

                // GPX / Drawing links
                if message.messageType == "gpx" {
                    HStack(spacing: 4) {
                        Image(systemName: "doc.text")
                            .font(.caption)
                        Text("View GPX on Map")
                            .font(.caption.weight(.medium))
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .foregroundStyle(isOwn ? .white.opacity(0.8) : .blue)
                }

                if message.messageType == "drawing", message.metadata?.drawingId != nil {
                    HStack(spacing: 4) {
                        Image(systemName: "pencil")
                            .font(.caption)
                        Text("View Drawing on Map")
                            .font(.caption.weight(.medium))
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .foregroundStyle(isOwn ? .white.opacity(0.8) : .blue)
                }
            }
            .background(isOwn ? Color.blue : Color(.systemGray5))
            .foregroundStyle(isOwn ? .white : .primary)
            .clipShape(RoundedRectangle(cornerRadius: 16))

            // Footer: time + location
            HStack(spacing: 4) {
                Text(formatTime(message.createdAt))
                    .font(.caption2)
                    .foregroundStyle(.secondary)

                if let lat = message.lat, let lng = message.lng {
                    HStack(spacing: 2) {
                        Image(systemName: "mappin")
                            .font(.system(size: 8))
                        Text(String(format: "%.4f, %.4f", lat, lng))
                    }
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                }
            }
        }
        .frame(maxWidth: UIScreen.main.bounds.width * 0.75, alignment: isOwn ? .trailing : .leading)
        .frame(maxWidth: .infinity, alignment: isOwn ? .trailing : .leading)
        .accessibilityElement(children: .combine)
        .accessibilityLabel(messageAccessibilityLabel)
    }

    private var messageAccessibilityLabel: String {
        var parts: [String] = []
        if !isOwn {
            parts.append("From \(message.displayName.isEmpty ? message.username : message.displayName)")
        }
        if !message.content.isEmpty {
            parts.append(message.content)
        }
        if !message.attachments.isEmpty {
            let count = message.attachments.count
            parts.append("\(count) attachment\(count == 1 ? "" : "s")")
        }
        parts.append(formatTime(message.createdAt))
        return parts.joined(separator: ". ")
    }

    // MARK: - Helpers

    @ViewBuilder
    private func fileAttachmentRow(_ attachment: Attachment) -> some View {
        HStack(spacing: 8) {
            Image(systemName: "arrow.down.doc")
                .font(.caption)
            Text(attachment.filename)
                .font(.caption)
                .lineLimit(1)
            Spacer()
            Text(attachment.formattedSize)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(isOwn ? Color.white.opacity(0.1) : Color(.systemGray6))
    }

    private func attachmentURL(for attachment: Attachment) -> URL? {
        guard let baseURL = KeychainStore.shared.serverURL,
              let token = KeychainStore.shared.accessToken
        else { return nil }

        let path = Endpoints.attachmentDownload(attachment.id)
        var components = URLComponents(string: "\(baseURL)\(path)")
        components?.queryItems = [URLQueryItem(name: "token", value: token)]
        return components?.url
    }

    private func formatTime(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        guard let date = formatter.date(from: iso) else { return "" }
        let timeFormatter = DateFormatter()
        timeFormatter.dateFormat = "HH:mm"
        return timeFormatter.string(from: date)
    }
}
