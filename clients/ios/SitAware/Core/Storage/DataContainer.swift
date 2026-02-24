import Foundation
import SwiftData

/// SwiftData container factory for the app's persistent models.
///
/// Configures the `ModelContainer` with all cached models and the offline action queue.
/// Used by `SitAwareApp` to inject SwiftData into the environment.
enum DataContainer {

    /// All SwiftData model types used by the app.
    static let modelTypes: [any PersistentModel.Type] = [
        CachedUser.self,
        CachedGroup.self,
        CachedMessage.self,
        CachedDrawing.self,
        CachedLocationEntry.self,
        OfflineAction.self,
    ]

    /// Create the shared model container.
    /// Uses the default SQLite store in the app's container directory.
    static func create() throws -> ModelContainer {
        let schema = Schema(modelTypes)
        let config = ModelConfiguration(
            schema: schema,
            isStoredInMemoryOnly: false,
            allowsSave: true)
        return try ModelContainer(for: schema, configurations: [config])
    }

    /// Create an in-memory container for previews and testing.
    static func createPreview() throws -> ModelContainer {
        let schema = Schema(modelTypes)
        let config = ModelConfiguration(
            schema: schema,
            isStoredInMemoryOnly: true)
        return try ModelContainer(for: schema, configurations: [config])
    }
}
