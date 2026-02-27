import SwiftUI

/// Admin map configuration — API keys, tile configs, terrain configs.
///
/// Mirrors the web client's `settings/server/map/page.tsx`:
/// - API keys section (MapBox, Google Maps)
/// - Tile configs CRUD (create, edit, enable/disable, set default, delete)
/// - Terrain configs CRUD
struct AdminMapConfigView: View {
    // API Keys
    @State private var mapboxToken = ""
    @State private var googleApiKey = ""
    @State private var isSavingKeys = false

    // Tile configs
    @State private var tileConfigs: [MapConfigResponse] = []
    @State private var isLoadingTiles = true

    // Terrain configs
    @State private var terrainConfigs: [TerrainConfigResponse] = []
    @State private var isLoadingTerrain = true

    @State private var errorMessage: String?
    @State private var successMessage: String?

    // Create/edit tile config
    @State private var showTileSheet = false
    @State private var editingTileConfig: MapConfigResponse?
    @State private var tileName = ""
    @State private var tileSourceType = "remote"
    @State private var tileUrl = ""
    @State private var tileMinZoom = 0
    @State private var tileMaxZoom = 22
    @State private var tileIsDefault = false
    @State private var isSavingTile = false

    // Create/edit terrain config
    @State private var showTerrainSheet = false
    @State private var editingTerrainConfig: TerrainConfigResponse?
    @State private var terrainName = ""
    @State private var terrainSourceType = "remote"
    @State private var terrainUrl = ""
    @State private var terrainEncoding = "terrarium"
    @State private var terrainIsDefault = false
    @State private var isSavingTerrain = false

    private let api = APIClient.shared

    var body: some View {
        List {
            // MARK: - API Keys
            Section("API Keys") {
                SecureField("MapBox Access Token", text: $mapboxToken)
                SecureField("Google Maps API Key", text: $googleApiKey)

                Button {
                    Task { await saveAPIKeys() }
                } label: {
                    if isSavingKeys {
                        ProgressView().frame(maxWidth: .infinity)
                    } else {
                        Text("Save Keys").frame(maxWidth: .infinity)
                    }
                }
                .buttonStyle(.borderedProminent)
                .disabled(isSavingKeys)
            }

            // MARK: - Tile Configs
            Section {
                if isLoadingTiles {
                    ProgressView()
                } else if tileConfigs.isEmpty {
                    Text("No tile configurations")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(tileConfigs, id: \.id) { config in
                        tileConfigRow(config)
                    }
                }
            } header: {
                HStack {
                    Text("Tile Configurations")
                    Spacer()
                    Button("Add") {
                        resetTileForm()
                        editingTileConfig = nil
                        showTileSheet = true
                    }
                    .font(.caption)
                }
            }

            // MARK: - Terrain Configs
            Section {
                if isLoadingTerrain {
                    ProgressView()
                } else if terrainConfigs.isEmpty {
                    Text("No terrain configurations")
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(terrainConfigs, id: \.id) { config in
                        terrainConfigRow(config)
                    }
                }
            } header: {
                HStack {
                    Text("Terrain Configurations")
                    Spacer()
                    Button("Add") {
                        resetTerrainForm()
                        editingTerrainConfig = nil
                        showTerrainSheet = true
                    }
                    .font(.caption)
                }
            }

            if let error = errorMessage {
                Section { Text(error).foregroundStyle(.red).font(.caption) }
            }
            if let success = successMessage {
                Section { Text(success).foregroundStyle(.green).font(.caption) }
            }
        }
        .navigationTitle("Map Configuration")
        .task { await loadAll() }
        .refreshable { await loadAll() }
        .sheet(isPresented: $showTileSheet) { tileConfigSheet }
        .sheet(isPresented: $showTerrainSheet) { terrainConfigSheet }
    }

    // MARK: - Tile Config Row

    @ViewBuilder
    private func tileConfigRow(_ config: MapConfigResponse) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(config.name)
                    .font(.subheadline.weight(.medium))

                if config.isBuiltin {
                    Text("Built-in")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.secondary.opacity(0.15))
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                }

                if config.isDefault {
                    Text("Default")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.green.opacity(0.15))
                        .foregroundStyle(.green)
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                }

                if !config.isEnabled {
                    Text("Disabled")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.red.opacity(0.15))
                        .foregroundStyle(.red)
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                }

                Spacer()
            }

            if !config.tileUrl.isEmpty {
                Text(config.tileUrl)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }

            HStack(spacing: 12) {
                if !(config.isDefault) && !(!config.isEnabled) {
                    Button("Set Default") {
                        // Would need an endpoint — using update
                        Task { await setTileDefault(config) }
                    }
                    .font(.caption)
                }

                if !(config.isBuiltin) {
                    Button("Edit") {
                        editingTileConfig = config
                        tileName = config.name
                        tileSourceType = config.sourceType
                        tileUrl = config.tileUrl
                        tileMinZoom = config.minZoom
                        tileMaxZoom = config.maxZoom
                        tileIsDefault = config.isDefault
                        showTileSheet = true
                    }
                    .font(.caption)
                }

                Spacer()

                if !(config.isBuiltin) {
                    Button(role: .destructive) {
                        Task { await deleteTileConfig(config.id) }
                    } label: {
                        Label("Delete", systemImage: "trash")
                            .font(.caption)
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }

    // MARK: - Terrain Config Row

    @ViewBuilder
    private func terrainConfigRow(_ config: TerrainConfigResponse) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(config.name)
                    .font(.subheadline.weight(.medium))

                if config.isDefault {
                    Text("Default")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.green.opacity(0.15))
                        .foregroundStyle(.green)
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                }

                Spacer()
            }

            if !config.terrainUrl.isEmpty {
                Text(config.terrainUrl)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }

            Text("Encoding: \(config.terrainEncoding)")
                .font(.caption2)
                .foregroundStyle(.secondary)

            HStack(spacing: 12) {
                if !(config.isBuiltin) {
                    Button("Edit") {
                        editingTerrainConfig = config
                        terrainName = config.name
                        terrainSourceType = config.sourceType
                        terrainUrl = config.terrainUrl
                        terrainEncoding = config.terrainEncoding
                        terrainIsDefault = config.isDefault
                        showTerrainSheet = true
                    }
                    .font(.caption)
                }

                Spacer()

                if !(config.isBuiltin) {
                    Button(role: .destructive) {
                        Task { await deleteTerrainConfig(config.id) }
                    } label: {
                        Label("Delete", systemImage: "trash")
                            .font(.caption)
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }

    // MARK: - Tile Config Sheet

    @ViewBuilder
    private var tileConfigSheet: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Name", text: $tileName)
                    Picker("Source Type", selection: $tileSourceType) {
                        Text("Remote").tag("remote")
                        Text("Local (MinIO)").tag("local")
                        Text("Style JSON").tag("style")
                    }
                    TextField("Tile URL", text: $tileUrl)
                        .textInputAutocapitalization(.never)
                    Stepper("Min Zoom: \(tileMinZoom)", value: $tileMinZoom, in: 0...24)
                    Stepper("Max Zoom: \(tileMaxZoom)", value: $tileMaxZoom, in: 0...24)
                    Toggle("Set as Default", isOn: $tileIsDefault)
                }
            }
            .navigationTitle(editingTileConfig == nil ? "Add Tile Config" : "Edit Tile Config")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showTileSheet = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await saveTileConfig() }
                    }
                    .disabled(tileName.isEmpty || tileUrl.isEmpty || isSavingTile)
                }
            }
        }
    }

    // MARK: - Terrain Config Sheet

    @ViewBuilder
    private var terrainConfigSheet: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Name", text: $terrainName)
                    Picker("Source Type", selection: $terrainSourceType) {
                        Text("Remote").tag("remote")
                        Text("Local (MinIO)").tag("local")
                    }
                    TextField("Terrain URL (DEM tile URL)", text: $terrainUrl)
                        .textInputAutocapitalization(.never)
                    Picker("Encoding", selection: $terrainEncoding) {
                        Text("Terrarium").tag("terrarium")
                        Text("Mapbox").tag("mapbox")
                    }
                    Toggle("Set as Default", isOn: $terrainIsDefault)
                }
            }
            .navigationTitle(editingTerrainConfig == nil ? "Add Terrain Config" : "Edit Terrain Config")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showTerrainSheet = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await saveTerrainConfig() }
                    }
                    .disabled(terrainName.isEmpty || terrainUrl.isEmpty || isSavingTerrain)
                }
            }
        }
    }

    // MARK: - API Actions

    private func loadAll() async {
        await withTaskGroup(of: Void.self) { group in
            group.addTask { await loadServerSettings() }
            group.addTask { await loadTileConfigs() }
            group.addTask { await loadTerrainConfigs() }
        }
    }

    private func loadServerSettings() async {
        do {
            let settings: ServerSettings = try await api.get(Endpoints.serverSettings)
            mapboxToken = settings.mapboxAccessToken
            googleApiKey = settings.googleMapsApiKey
        } catch {
            // Non-critical — keys remain empty
        }
    }

    private func loadTileConfigs() async {
        isLoadingTiles = true
        do {
            tileConfigs = try await api.get(Endpoints.mapConfigs)
        } catch {
            errorMessage = "Failed to load tile configs"
        }
        isLoadingTiles = false
    }

    private func loadTerrainConfigs() async {
        isLoadingTerrain = true
        do {
            terrainConfigs = try await api.get(Endpoints.terrainConfigs)
        } catch {
            errorMessage = "Failed to load terrain configs"
        }
        isLoadingTerrain = false
    }

    private func saveAPIKeys() async {
        isSavingKeys = true
        errorMessage = nil
        successMessage = nil

        do {
            struct KeysBody: Encodable {
                let mapboxAccessToken: String
                let googleMapsApiKey: String
            }
            let _: ServerSettings = try await api.put(
                Endpoints.serverSettings,
                body: KeysBody(mapboxAccessToken: mapboxToken, googleMapsApiKey: googleApiKey))
            successMessage = "API keys saved"
        } catch {
            errorMessage = "Failed to save API keys"
        }

        isSavingKeys = false
    }

    private func saveTileConfig() async {
        isSavingTile = true
        errorMessage = nil

        do {
            let body = CreateMapConfigRequest(
                name: tileName,
                sourceType: tileSourceType,
                tileUrl: tileUrl,
                minZoom: tileMinZoom,
                maxZoom: tileMaxZoom,
                isDefault: tileIsDefault)

            if let existing = editingTileConfig {
                let updateBody = UpdateMapConfigRequest(
                    name: tileName,
                    sourceType: tileSourceType,
                    tileUrl: tileUrl,
                    minZoom: tileMinZoom,
                    maxZoom: tileMaxZoom,
                    isDefault: tileIsDefault)
                let _: MapConfigResponse = try await api.put(
                    Endpoints.mapConfig(existing.id), body: updateBody)
            } else {
                let _: MapConfigResponse = try await api.post(Endpoints.mapConfigs, body: body)
            }

            showTileSheet = false
            await loadTileConfigs()
        } catch {
            errorMessage = "Failed to save tile config"
        }

        isSavingTile = false
    }

    private func deleteTileConfig(_ id: String) async {
        do {
            try await api.delete(Endpoints.mapConfig(id))
            await loadTileConfigs()
        } catch {
            errorMessage = "Failed to delete tile config"
        }
    }

    private func setTileDefault(_ config: MapConfigResponse) async {
        do {
            let body = UpdateMapConfigRequest(isDefault: true)
            let _: MapConfigResponse = try await api.put(Endpoints.mapConfig(config.id), body: body)
            await loadTileConfigs()
        } catch {
            errorMessage = "Failed to set default"
        }
    }

    private func saveTerrainConfig() async {
        isSavingTerrain = true
        errorMessage = nil

        do {
            struct TerrainBody: Encodable {
                let name: String
                let sourceType: String
                let terrainUrl: String
                let terrainEncoding: String
                let isDefault: Bool
            }
            let body = TerrainBody(
                name: terrainName, sourceType: terrainSourceType,
                terrainUrl: terrainUrl, terrainEncoding: terrainEncoding,
                isDefault: terrainIsDefault)

            if let existing = editingTerrainConfig {
                let _: TerrainConfigResponse = try await api.put(
                    Endpoints.terrainConfig(existing.id), body: body)
            } else {
                let _: TerrainConfigResponse = try await api.post(Endpoints.terrainConfigs, body: body)
            }

            showTerrainSheet = false
            await loadTerrainConfigs()
        } catch {
            errorMessage = "Failed to save terrain config"
        }

        isSavingTerrain = false
    }

    private func deleteTerrainConfig(_ id: String) async {
        do {
            try await api.delete(Endpoints.terrainConfig(id))
            await loadTerrainConfigs()
        } catch {
            errorMessage = "Failed to delete terrain config"
        }
    }

    private func resetTileForm() {
        tileName = ""
        tileSourceType = "remote"
        tileUrl = ""
        tileMinZoom = 0
        tileMaxZoom = 22
        tileIsDefault = false
    }

    private func resetTerrainForm() {
        terrainName = ""
        terrainSourceType = "remote"
        terrainUrl = ""
        terrainEncoding = "terrarium"
        terrainIsDefault = false
    }
}
