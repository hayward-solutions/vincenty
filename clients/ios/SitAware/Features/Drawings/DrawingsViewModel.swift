import Foundation

/// Draw mode: the type of shape being drawn.
enum DrawMode: String, CaseIterable, Sendable {
    case line
    case circle
    case rectangle
}

/// Style for the current drawing tool.
struct DrawStyle: Sendable, Equatable {
    var stroke: String = "#3b82f6"
    var fill: String = "transparent"
    var strokeWidth: Double = 2

    static let strokePresets = [
        "#ef4444", "#f97316", "#eab308", "#22c55e", "#06b6d4",
        "#3b82f6", "#8b5cf6", "#ec4899", "#ffffff", "#000000",
    ]
    static let fillPresets = [
        "transparent", "#ef4444", "#f97316", "#eab308", "#22c55e",
        "#06b6d4", "#3b82f6", "#8b5cf6", "#ec4899", "#000000",
    ]
}

/// A completed shape from the draw tool (not yet saved).
struct CompletedShape: Identifiable, Sendable {
    let id = UUID()
    let feature: GeoJSONFeature
    let shapeType: DrawMode
}

/// Manages drawing CRUD, sharing, visibility, and WS updates.
///
/// Mirrors the web client's `use-drawings.ts` combined hook:
/// - Own drawings + shared drawings fetched separately
/// - WebSocket `drawing_updated` updates in-place
/// - WebSocket `message_new` with `message_type == "drawing"` triggers shared refetch
/// - Per-drawing visibility toggling via `hiddenDrawingIds`
@Observable @MainActor
final class DrawingsViewModel {

    // MARK: - Drawing Tool State

    var drawMode: DrawMode = .line
    var drawStyle = DrawStyle()
    var completedShapes: [CompletedShape] = []
    var drawingName = ""
    var savedDrawingId: String?

    // MARK: - Saved Drawings

    private(set) var ownDrawings: [DrawingResponse] = []
    private(set) var sharedDrawings: [DrawingResponse] = []
    private(set) var isLoadingDrawings = false

    // MARK: - Visibility

    /// IDs of drawings that are hidden (toggled off).
    var hiddenDrawingIds: Set<String> = []

    /// All visible drawings (own + shared, excluding hidden).
    var visibleDrawings: [DrawingResponse] {
        (ownDrawings + sharedDrawings).filter { !hiddenDrawingIds.contains($0.id) }
    }

    func toggleVisibility(_ drawingId: String) {
        if hiddenDrawingIds.contains(drawingId) {
            hiddenDrawingIds.remove(drawingId)
        } else {
            hiddenDrawingIds.insert(drawingId)
        }
    }

    // MARK: - Shares

    private(set) var drawingShares: [DrawingShareInfo] = []
    private(set) var drawingSharesLoading = false
    var managingDrawingId: String?

    // MARK: - Save/Share State

    private(set) var isSaving = false
    private(set) var isSharing = false

    // MARK: - Private

    private let api = APIClient.shared
    private var wsUnsubscribe: (() -> Void)?

    // MARK: - Fetch

    func loadDrawings() async {
        isLoadingDrawings = true
        defer { isLoadingDrawings = false }

        async let ownTask: [DrawingResponse] = {
            do { return try await api.get(Endpoints.drawings) }
            catch { return [] }
        }()
        async let sharedTask: [DrawingResponse] = {
            do { return try await api.get(Endpoints.drawingsShared) }
            catch { return [] }
        }()

        let (own, shared) = await (ownTask, sharedTask)
        ownDrawings = own
        sharedDrawings = shared
    }

    func loadShares(for drawingId: String) async {
        drawingSharesLoading = true
        defer { drawingSharesLoading = false }

        do {
            drawingShares = try await api.get(Endpoints.drawingShares(drawingId))
        } catch {
            drawingShares = []
        }
    }

    // MARK: - CRUD

    func saveDrawing() async throws {
        guard !drawingName.trimmingCharacters(in: .whitespaces).isEmpty else { return }

        isSaving = true
        defer { isSaving = false }

        let geojson = buildFeatureCollection()

        if let existingId = savedDrawingId {
            // Update
            let body = UpdateDrawingRequest(name: drawingName, geojson: geojson)
            let updated: DrawingResponse = try await api.put(
                Endpoints.drawing(existingId), body: body)
            // Update in local list
            if let idx = ownDrawings.firstIndex(where: { $0.id == existingId }) {
                ownDrawings[idx] = updated
            }
        } else {
            // Create
            let body = CreateDrawingRequest(name: drawingName, geojson: geojson)
            let created: DrawingResponse = try await api.post(Endpoints.drawings, body: body)
            savedDrawingId = created.id
            ownDrawings.insert(created, at: 0)
        }
    }

    func deleteDrawing(_ id: String) async throws {
        try await api.delete(Endpoints.drawing(id))
        ownDrawings.removeAll { $0.id == id }
        sharedDrawings.removeAll { $0.id == id }
        if savedDrawingId == id {
            savedDrawingId = nil
        }
    }

    // MARK: - Share / Unshare

    func shareDrawing(_ drawingId: String, groupId: String) async throws {
        isSharing = true
        defer { isSharing = false }

        let body = ShareDrawingRequest(groupId: groupId)
        try await api.post(Endpoints.drawingShare(drawingId), body: body) as EmptyResponse
    }

    func unshareDrawing(drawingId: String, messageId: String) async throws {
        try await api.delete(Endpoints.drawingUnshare(drawingId, messageId))
        drawingShares.removeAll { $0.messageId == messageId }
    }

    // MARK: - Draw Tool Actions

    func addCompletedShape(_ feature: GeoJSONFeature, type: DrawMode) {
        completedShapes.append(CompletedShape(feature: feature, shapeType: type))
    }

    func removeShape(at index: Int) {
        guard completedShapes.indices.contains(index) else { return }
        completedShapes.remove(at: index)
    }

    func clearShapes() {
        completedShapes.removeAll()
    }

    func resetDrawingTool() {
        completedShapes.removeAll()
        drawingName = ""
        savedDrawingId = nil
    }

    // MARK: - Build GeoJSON

    private func buildFeatureCollection() -> GeoJSONFeatureCollection {
        GeoJSONFeatureCollection(features: completedShapes.map(\.feature))
    }

    // MARK: - WebSocket

    func subscribeToUpdates(webSocket: WebSocketService, currentUserId: String?) {
        wsUnsubscribe = webSocket.subscribe { [weak self] type, payload in
            Task { @MainActor [weak self] in
                guard let self, let payload else { return }

                switch type {
                case WSMessageType.drawingUpdated:
                    if let data = try? JSONSerialization.data(withJSONObject: payload.value),
                       let updated = try? JSONDecoder.snakeCase.decode(
                        DrawingResponse.self, from: data)
                    {
                        if updated.ownerId == currentUserId {
                            if let idx = self.ownDrawings.firstIndex(where: { $0.id == updated.id }) {
                                self.ownDrawings[idx] = updated
                            }
                        } else {
                            if let idx = self.sharedDrawings.firstIndex(where: {
                                $0.id == updated.id
                            }) {
                                self.sharedDrawings[idx] = updated
                            } else {
                                self.sharedDrawings.insert(updated, at: 0)
                            }
                        }
                    }

                case WSMessageType.messageNew:
                    if let data = try? JSONSerialization.data(withJSONObject: payload.value),
                       let msg = try? JSONDecoder.snakeCase.decode(
                        MessageResponse.self, from: data)
                    {
                        if msg.messageType == "drawing" && msg.senderId != currentUserId {
                            // Refetch shared drawings
                            await self.loadDrawings()
                        }
                    }

                default:
                    break
                }
            }
        }
    }

    func unsubscribe() {
        wsUnsubscribe?()
        wsUnsubscribe = nil
    }
}
