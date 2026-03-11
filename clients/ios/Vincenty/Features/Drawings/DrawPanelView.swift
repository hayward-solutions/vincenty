import SwiftUI

/// Drawing panel shown when the draw tool is active.
///
/// Mirrors the web client's `draw-panel.tsx`:
/// - Tool mode selector (line/circle/rectangle)
/// - Stroke and fill color pickers
/// - In-session shape list with remove
/// - Name input + save/update button
/// - Saved drawings list with visibility toggles
struct DrawPanelView: View {
    @Bindable var viewModel: DrawingsViewModel
    let groups: [Group]
    let onClose: () -> Void

    @State private var showShareList = false
    @State private var shareError: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // Header
                HStack {
                    Text("Draw")
                        .font(.headline)
                    Spacer()
                    Button { onClose() } label: {
                        Image(systemName: "xmark")
                            .font(.subheadline)
                    }
                    .accessibilityLabel("Close draw panel")
                }

                // Tool mode selector
                HStack(spacing: 8) {
                    ForEach(DrawMode.allCases, id: \.rawValue) { mode in
                        Button {
                            viewModel.drawMode = mode
                        } label: {
                            VStack(spacing: 4) {
                                Image(systemName: iconForMode(mode))
                                    .font(.system(size: 18))
                                Text(mode.rawValue.capitalized)
                                    .font(.caption2)
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(
                                viewModel.drawMode == mode
                                    ? Color.accentColor.opacity(0.15)
                                    : Color.clear)
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                        }
                        .foregroundStyle(viewModel.drawMode == mode ? .primary : .secondary)
                    }
                }

                // Stroke color
                VStack(alignment: .leading, spacing: 4) {
                    Text("Stroke")
                        .font(.caption.weight(.medium))
                    colorRow(
                        colors: DrawStyle.strokePresets,
                        selected: viewModel.drawStyle.stroke
                    ) { viewModel.drawStyle.stroke = $0 }
                }

                // Fill color
                VStack(alignment: .leading, spacing: 4) {
                    Text("Fill")
                        .font(.caption.weight(.medium))
                    colorRow(
                        colors: DrawStyle.fillPresets,
                        selected: viewModel.drawStyle.fill
                    ) { viewModel.drawStyle.fill = $0 }
                }

                Divider()

                // In-session shapes
                if viewModel.completedShapes.isEmpty {
                    Text(instructionForMode(viewModel.drawMode))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity, alignment: .center)
                        .padding(.vertical, 8)
                } else {
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text("Shapes (\(viewModel.completedShapes.count))")
                                .font(.caption.weight(.medium))
                            Spacer()
                            Button("Clear All") { viewModel.clearShapes() }
                                .font(.caption)
                        }

                        ForEach(Array(viewModel.completedShapes.enumerated()), id: \.element.id) {
                            idx, shape in
                            HStack(spacing: 8) {
                                Circle()
                                    .fill(
                                        Color(
                                            hex: shape.feature.properties?["stroke"]?.value
                                                as? String ?? "#3b82f6") ?? .blue)
                                    .frame(width: 10, height: 10)
                                Text(shape.shapeType.rawValue.capitalized)
                                    .font(.caption)
                                Spacer()
                                Button {
                                    viewModel.removeShape(at: idx)
                                } label: {
                                    Image(systemName: "trash")
                                        .font(.caption)
                                        .foregroundStyle(.red)
                                }
                            }
                            .padding(.vertical, 2)
                        }
                    }
                }

                Divider()

                // Save section
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Drawing name", text: $viewModel.drawingName)
                        .textFieldStyle(.roundedBorder)
                        .font(.subheadline)

                    Button {
                        Task { try? await viewModel.saveDrawing() }
                    } label: {
                        if viewModel.isSaving {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                        } else {
                            Text(viewModel.savedDrawingId != nil ? "Update" : "Save")
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(
                        viewModel.completedShapes.isEmpty
                            || viewModel.drawingName.trimmingCharacters(in: .whitespaces).isEmpty
                            || viewModel.isSaving)
                }

                // Saved drawings list
                if !viewModel.ownDrawings.isEmpty || !viewModel.sharedDrawings.isEmpty {
                    Divider()

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Saved Drawings")
                            .font(.caption.weight(.semibold))

                        // Own drawings
                        if !viewModel.ownDrawings.isEmpty {
                            Text("Mine")
                                .font(.caption2.weight(.medium))
                                .foregroundStyle(.secondary)

                            ForEach(viewModel.ownDrawings) { drawing in
                                drawingRow(drawing, isOwn: true)
                            }
                        }

                        // Shared drawings
                        if !viewModel.sharedDrawings.isEmpty {
                            Text("Shared with me")
                                .font(.caption2.weight(.medium))
                                .foregroundStyle(.secondary)

                            ForEach(viewModel.sharedDrawings) { drawing in
                                drawingRow(drawing, isOwn: false)
                            }
                        }
                    }
                }
            }
            .padding(12)
        }
        .frame(width: 280, height: UIScreen.main.bounds.height * 0.7)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .shadow(color: .black.opacity(0.15), radius: 4, x: 0, y: 2)
    }

    // MARK: - Drawing Row

    @ViewBuilder
    private func drawingRow(_ drawing: DrawingResponse, isOwn: Bool) -> some View {
        HStack(spacing: 8) {
            Button {
                viewModel.toggleVisibility(drawing.id)
            } label: {
                Image(
                    systemName: viewModel.hiddenDrawingIds.contains(drawing.id)
                        ? "eye.slash" : "eye")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .accessibilityLabel(viewModel.hiddenDrawingIds.contains(drawing.id) ? "Show drawing" : "Hide drawing")
            .accessibilityValue(drawing.name)

            Text(drawing.name)
                .font(.caption)
                .lineLimit(1)

            if !isOwn {
                Text(drawing.displayName.isEmpty ? drawing.username : drawing.displayName)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Spacer()

            if isOwn {
                Button {
                    Task { try? await viewModel.deleteDrawing(drawing.id) }
                } label: {
                    Image(systemName: "trash")
                        .font(.caption)
                        .foregroundStyle(.red)
                }
                .accessibilityLabel("Delete \(drawing.name)")
            }
        }
        .padding(.vertical, 2)
    }

    // MARK: - Color Picker

    @ViewBuilder
    private func colorRow(
        colors: [String], selected: String, onSelect: @escaping (String) -> Void
    ) -> some View {
        HStack(spacing: 4) {
            ForEach(colors, id: \.self) { color in
                Button {
                    onSelect(color)
                } label: {
                    if color == "transparent" {
                        // Checkerboard pattern for transparent
                        RoundedRectangle(cornerRadius: 4)
                            .strokeBorder(.secondary, lineWidth: 1)
                            .frame(width: 22, height: 22)
                            .overlay(
                                Image(systemName: "line.diagonal")
                                    .font(.system(size: 10))
                                    .foregroundStyle(.secondary))
                    } else {
                        RoundedRectangle(cornerRadius: 4)
                            .fill(Color(hex: color))
                            .frame(width: 22, height: 22)
                    }
                }
                .overlay(
                    RoundedRectangle(cornerRadius: 4)
                        .strokeBorder(
                            selected == color ? Color.primary : .clear,
                            lineWidth: 2))
                .accessibilityLabel(color == "transparent" ? "Transparent" : "Color \(color)")
                .accessibilityValue(selected == color ? "Selected" : "")
            }
        }
    }

    // MARK: - Helpers

    private func iconForMode(_ mode: DrawMode) -> String {
        switch mode {
        case .line: return "line.diagonal"
        case .circle: return "circle"
        case .rectangle: return "rectangle"
        }
    }

    private func instructionForMode(_ mode: DrawMode) -> String {
        switch mode {
        case .line: return "Tap points on the map, double-tap to finish"
        case .circle: return "Tap center, then tap edge"
        case .rectangle: return "Tap three corners to define the rectangle"
        }
    }
}
