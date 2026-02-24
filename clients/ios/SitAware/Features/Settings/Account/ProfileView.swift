import PhotosUI
import SwiftUI

/// Account profile settings — avatar, map marker, display name, email.
///
/// Mirrors the web client's `settings/account/general/page.tsx`:
/// - Avatar upload/remove
/// - Map marker icon + color picker
/// - Display name and email editing
struct ProfileView: View {
    @Environment(AuthManager.self) private var auth

    @State private var displayName = ""
    @State private var email = ""
    @State private var markerIcon = "default"
    @State private var markerColor = "#3b82f6"

    // Avatar
    @State private var avatarItem: PhotosPickerItem?
    @State private var isUploadingAvatar = false
    @State private var isRemovingAvatar = false

    // Profile save
    @State private var isSaving = false
    @State private var isMarkerSaving = false
    @State private var errorMessage: String?
    @State private var successMessage: String?

    private let api = APIClient.shared

    // MARK: - Marker Shape Presets

    static let availableShapes = [
        "default", "circle", "square", "diamond", "triangle",
        "star", "hexagon", "pentagon", "cross", "shield",
    ]

    static let presetColors = [
        "#ef4444", "#f97316", "#eab308", "#22c55e", "#06b6d4",
        "#3b82f6", "#8b5cf6", "#ec4899", "#ffffff", "#000000",
    ]

    var body: some View {
        Form {
            // MARK: - Avatar Section
            Section("Avatar") {
                HStack(spacing: 16) {
                    avatarView

                    VStack(alignment: .leading, spacing: 8) {
                        PhotosPicker(selection: $avatarItem, matching: .images) {
                            Label("Upload", systemImage: "arrow.up.circle")
                                .font(.subheadline)
                        }
                        .disabled(isUploadingAvatar)

                        if let user = auth.user, !user.avatarUrl.isEmpty {
                            Button(role: .destructive) {
                                Task { await removeAvatar() }
                            } label: {
                                Label("Remove", systemImage: "trash")
                                    .font(.subheadline)
                            }
                            .disabled(isRemovingAvatar)
                        }
                    }
                }
            }

            // MARK: - Map Marker Section
            Section("Map Marker") {
                VStack(alignment: .leading, spacing: 12) {
                    // Shape picker
                    Text("Shape")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)

                    LazyVGrid(columns: Array(repeating: GridItem(.flexible()), count: 5), spacing: 8) {
                        ForEach(Self.availableShapes, id: \.self) { shape in
                            Button {
                                markerIcon = shape
                            } label: {
                                Text(labelForShape(shape))
                                    .font(.system(size: 11))
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 8)
                                    .background(
                                        markerIcon == shape
                                            ? Color.accentColor.opacity(0.15)
                                            : Color(.secondarySystemGroupedBackground))
                                    .clipShape(RoundedRectangle(cornerRadius: 8))
                            }
                            .foregroundStyle(markerIcon == shape ? .primary : .secondary)
                            .accessibilityLabel("\(labelForShape(shape)) marker shape")
                            .accessibilityValue(markerIcon == shape ? "Selected" : "")
                        }
                    }

                    // Color picker
                    Text("Color")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.secondary)

                    HStack(spacing: 6) {
                        ForEach(Self.presetColors, id: \.self) { color in
                            Button {
                                markerColor = color
                            } label: {
                                RoundedRectangle(cornerRadius: 4)
                                    .fill(Color(hex: color) ?? .blue)
                                    .frame(width: 26, height: 26)
                                    .overlay(
                                        RoundedRectangle(cornerRadius: 4)
                                            .strokeBorder(
                                                markerColor == color ? Color.primary : .clear,
                                                lineWidth: 2))
                            }
                            .accessibilityLabel("Marker color \(colorName(for: color))")
                            .accessibilityValue(markerColor == color ? "Selected" : "")
                        }
                    }
                }

                Button {
                    Task { await saveMarker() }
                } label: {
                    if isMarkerSaving {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                    } else {
                        Text("Save Marker")
                            .frame(maxWidth: .infinity)
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(isMarkerSaving)
            }

            // MARK: - Profile Section
            Section("Profile") {
                LabeledContent("Username") {
                    Text(auth.user?.username ?? "")
                        .foregroundStyle(.secondary)
                }

                TextField("Display Name", text: $displayName)

                TextField("Email", text: $email)
                    .keyboardType(.emailAddress)
                    .textContentType(.emailAddress)
                    .autocapitalization(.none)

                Button {
                    Task { await saveProfile() }
                } label: {
                    if isSaving {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                    } else {
                        Text("Save Changes")
                            .frame(maxWidth: .infinity)
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(isSaving)
            }

            // MARK: - Status Messages
            if let error = errorMessage {
                Section {
                    Text(error)
                        .foregroundStyle(.red)
                        .font(.caption)
                }
            }

            if let success = successMessage {
                Section {
                    Text(success)
                        .foregroundStyle(.green)
                        .font(.caption)
                }
            }
        }
        .navigationTitle("General")
        .onAppear { loadFromUser() }
        .onChange(of: avatarItem) { _, newItem in
            if let item = newItem {
                Task { await uploadAvatar(item: item) }
            }
        }
    }

    // MARK: - Avatar View

    @ViewBuilder
    private var avatarView: some View {
        if let user = auth.user, !user.avatarUrl.isEmpty {
            AsyncImage(url: avatarURL(for: user)) { image in
                image
                    .resizable()
                    .scaledToFill()
            } placeholder: {
                initialsView
            }
            .frame(width: 80, height: 80)
            .clipShape(Circle())
            .accessibilityLabel("Profile avatar")
        } else {
            initialsView
                .accessibilityLabel("Profile avatar placeholder, showing initials \(initials)")
        }
    }

    private var initialsView: some View {
        Circle()
            .fill(Color.accentColor.opacity(0.2))
            .frame(width: 80, height: 80)
            .overlay(
                Text(initials)
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(.accentColor))
    }

    private var initials: String {
        guard let user = auth.user else { return "?" }
        let name = user.displayName.isEmpty ? user.username : user.displayName
        let parts = name.split(separator: " ")
        if parts.count >= 2 {
            return "\(parts[0].prefix(1))\(parts[1].prefix(1))".uppercased()
        }
        return String(name.prefix(2)).uppercased()
    }

    private func avatarURL(for user: User) -> URL? {
        guard let baseURL = KeychainStore.shared.serverURL,
              let token = KeychainStore.shared.accessToken
        else { return nil }
        return URL(string: "\(baseURL)\(Endpoints.userAvatar(user.id))?token=\(token)")
    }

    // MARK: - Actions

    private func loadFromUser() {
        guard let user = auth.user else { return }
        displayName = user.displayName
        email = user.email
        markerIcon = user.markerIcon.isEmpty ? "default" : user.markerIcon
        markerColor = user.markerColor.isEmpty ? "#3b82f6" : user.markerColor
    }

    private func uploadAvatar(item: PhotosPickerItem) async {
        isUploadingAvatar = true
        errorMessage = nil

        do {
            guard let data = try await item.loadTransferable(type: Data.self) else {
                errorMessage = "Could not load image data"
                isUploadingAvatar = false
                return
            }

            // Validate size (5MB limit)
            guard data.count <= 5 * 1024 * 1024 else {
                errorMessage = "Image must be less than 5MB"
                isUploadingAvatar = false
                return
            }

            var formData = MultipartFormData()
            formData.addFile(name: "avatar", filename: "avatar.jpg", data: data, contentType: "image/jpeg")

            let _: User = try await api.upload(Endpoints.usersMeAvatar, formData: formData)
            await auth.refreshUser()
        } catch {
            errorMessage = "Upload failed: \(error.localizedDescription)"
        }

        avatarItem = nil
        isUploadingAvatar = false
    }

    private func removeAvatar() async {
        isRemovingAvatar = true
        errorMessage = nil

        do {
            try await api.delete(Endpoints.usersMeAvatar)
            await auth.refreshUser()
        } catch {
            errorMessage = "Failed to remove avatar: \(error.localizedDescription)"
        }

        isRemovingAvatar = false
    }

    private func saveMarker() async {
        isMarkerSaving = true
        errorMessage = nil
        successMessage = nil

        do {
            let body = UpdateMeRequest(markerIcon: markerIcon, markerColor: markerColor)
            let _: User = try await api.put(Endpoints.usersMe, body: body)
            await auth.refreshUser()
            successMessage = "Marker updated"
        } catch {
            errorMessage = "Failed to save marker: \(error.localizedDescription)"
        }

        isMarkerSaving = false
    }

    private func saveProfile() async {
        isSaving = true
        errorMessage = nil
        successMessage = nil

        do {
            let body = UpdateMeRequest(
                email: email.isEmpty ? nil : email,
                displayName: displayName.isEmpty ? nil : displayName)
            let _: User = try await api.put(Endpoints.usersMe, body: body)
            await auth.refreshUser()
            successMessage = "Profile updated"
        } catch {
            errorMessage = "Failed to save: \(error.localizedDescription)"
        }

        isSaving = false
    }

    // MARK: - Helpers

    private func colorName(for hex: String) -> String {
        switch hex {
        case "#ef4444": return "Red"
        case "#f97316": return "Orange"
        case "#eab308": return "Yellow"
        case "#22c55e": return "Green"
        case "#06b6d4": return "Cyan"
        case "#3b82f6": return "Blue"
        case "#8b5cf6": return "Purple"
        case "#ec4899": return "Pink"
        case "#ffffff": return "White"
        case "#000000": return "Black"
        default: return hex
        }
    }

    private func labelForShape(_ shape: String) -> String {
        switch shape {
        case "default": return "Pin"
        case "circle": return "Circle"
        case "square": return "Square"
        case "diamond": return "Diamond"
        case "triangle": return "Triangle"
        case "star": return "Star"
        case "hexagon": return "Hexagon"
        case "pentagon": return "Pentagon"
        case "cross": return "Cross"
        case "shield": return "Shield"
        default: return shape.capitalized
        }
    }
}
