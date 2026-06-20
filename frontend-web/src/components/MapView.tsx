import { Map, Marker, NavigationControl } from "react-map-gl/maplibre";

const MAP_STYLE = "https://demotiles.maplibre.org/style.json";

interface MapViewProps {
  center?: [number, number]; // Defaults to Buenos Aires
  zoom?: number;
  markerLabel?: string;
}

export function MapView({ center = [-58.3816, -34.6037], zoom = 12, markerLabel }: MapViewProps) {
  return (
    <div className="h-96 w-full overflow-hidden rounded-lg border border-slate-700">
      <Map
        initialViewState={{
          longitude: center[0],
          latitude: center[1],
          zoom,
        }}
        style={{ width: "100%", height: "100%" }}
        mapStyle={MAP_STYLE}
      >
        <NavigationControl position="top-right" />
        <Marker longitude={center[0]} latitude={center[1]}>
          <div className="flex h-6 w-6 cursor-pointer items-center justify-center rounded-full border-2 border-white bg-blue-500 text-xs font-bold text-white shadow-lg">
            {markerLabel ?? "🚗"}
          </div>
        </Marker>
      </Map>
    </div>
  );
}
