import { useEffect, useRef, useState } from "react";
import cytoscape, { type Core, type LayoutOptions } from "cytoscape";
import { Maximize, Minus, Plus } from "lucide-react";
import { Button } from "../ui";
import type { GraphEdge, GraphNode } from "./model";

function fitGraph(cy: Core) {
  cy.fit(undefined, 48);
  // Small local graphs retain natural-sized nodes; manual zoom still allows 3x.
  if (cy.zoom() > 1.25) {
    cy.zoom(1.25);
    cy.center();
  }
}

export default function GraphCanvas({
  nodes,
  edges,
  selected,
  focus,
  layout,
  theme,
  select,
  open,
}: {
  nodes: GraphNode[];
  edges: GraphEdge[];
  selected: string;
  focus: string;
  layout: string;
  theme: string;
  select: (id: string) => void;
  open: (id: string) => void;
}) {
  const host = useRef<HTMLDivElement>(null),
    core = useRef<Core | null>(null),
    callbacks = useRef({ select, open });
  const [zoom, setZoom] = useState(100);
  callbacks.current = { select, open };
  const graphKey = JSON.stringify({
    nodes: nodes.map(({ id, title, indexed }) => ({ id, title, indexed })),
    edges,
  });
  useEffect(() => {
    if (!host.current || !nodes.length) return;
    const dark = theme === "dark";
    const cy = cytoscape({
      container: host.current,
      elements: [
        ...nodes.map((n) => ({
          data: {
            id: n.id,
            title: n.title,
            pending: n.indexed === false ? 1 : 0,
            focus: n.id === focus ? 1 : 0,
          },
        })),
        ...edges.map((e, index) => ({
          data: {
            id: `edge-${index}`,
            source: e.source,
            target: e.target,
            type: e.type,
          },
        })),
      ],
      minZoom: 0.08,
      maxZoom: 3,
      wheelSensitivity: 0.25,
      boxSelectionEnabled: false,
      style: [
        {
          selector: "node",
          style: {
            label: "data(title)",
            width: 42,
            height: 42,
            "background-color": dark ? "#376553" : "#dfede4",
            "border-color": "#719d85",
            "border-width": 2,
            color: dark ? "#e1ebe4" : "#283b34",
            "font-family": "Noto Sans KR Variable",
            "font-size": 15,
            "text-valign": "bottom",
            "text-margin-y": 10,
            "text-wrap": "ellipsis",
            "text-max-width": "150px",
            "min-zoomed-font-size": 12,
            "text-background-color": dark ? "#24362e" : "#f7f9f5",
            "text-background-opacity": 0.95,
            "text-background-padding": "3px",
          },
        },
        {
          selector: "node[focus = 1]",
          style: {
            width: 58,
            height: 58,
            "background-color": "#c5deb1",
            "border-color": "#638650",
            "border-width": 3,
          },
        },
        {
          selector: "node[pending = 1]",
          style: { "border-style": "dashed", "background-opacity": 0.6 },
        },
        {
          selector: "node:selected",
          style: {
            "border-width": 4,
            "border-color": dark ? "#a9dcbe" : "#196b55",
            "background-color": dark ? "#477e67" : "#c8e2d2",
            "font-weight": 700,
          },
        },
        {
          selector: "edge",
          style: {
            width: 1.8,
            "line-color": dark ? "#668779" : "#b0c5b6",
            "target-arrow-color": dark ? "#668779" : "#b0c5b6",
            "target-arrow-shape": "triangle",
            "curve-style": "bezier",
            "arrow-scale": 0.8,
          },
        },
        {
          selector: "edge[type = 'related']",
          style: {
            "line-color": "#a89bc5",
            "target-arrow-shape": "none",
            "line-style": "dashed",
          },
        },
        {
          selector: "edge[type = 'parent']",
          style: {
            "line-color": "#d6b882",
            "target-arrow-color": "#d6b882",
            "line-style": "dotted",
          },
        },
      ],
      layout: { name: "preset" },
    });
    core.current = cy;
    cy.on("tap", "node", (event) =>
      callbacks.current.select(event.target.id()),
    );
    cy.on("dbltap", "node", (event) =>
      callbacks.current.open(event.target.id()),
    );
    cy.on("zoom", () => setZoom(Math.round(cy.zoom() * 100)));
    const actualLayout =
      layout === "cose" && nodes.length > 300 ? "concentric" : layout;
    const options: LayoutOptions =
      actualLayout === "cose"
        ? {
            name: "cose",
            animate: false,
            numIter: 250,
            randomize: true,
            nodeRepulsion: () => 14000,
            idealEdgeLength: () => 120,
            nodeOverlap: 30,
            componentSpacing: 100,
            padding: 55,
          }
        : actualLayout === "breadthfirst"
          ? {
              name: "breadthfirst",
              directed: false,
              circle: false,
              ...(focus ? { roots: [focus] } : {}),
              spacingFactor: 1.4,
              padding: 55,
              animate: false,
              nodeDimensionsIncludeLabels: true,
            }
          : {
              name: "concentric",
              concentric: (node) =>
                node.id() === focus ? 100000 : node.degree(),
              levelWidth: () => 2,
              minNodeSpacing: 50,
              padding: 55,
              animate: false,
              nodeDimensionsIncludeLabels: true,
            };
    const arrangement = cy.layout(options);
    cy.one("layoutstop", () => fitGraph(cy));
    arrangement.run();
    let fittedWidth = cy.width(),
      fittedHeight = cy.height();
    const resize = new ResizeObserver(() => {
      cy.resize();
      if (
        Math.abs(cy.width() - fittedWidth) > 40 ||
        Math.abs(cy.height() - fittedHeight) > 60
      ) {
        fitGraph(cy);
        fittedWidth = cy.width();
        fittedHeight = cy.height();
      }
    });
    resize.observe(host.current);
    return () => {
      resize.disconnect();
      arrangement.stop();
      core.current = null;
      cy.destroy();
    };
  }, [graphKey, focus, layout, theme]);
  useEffect(() => {
    const cy = core.current;
    if (!cy) return;
    cy.elements().unselect();
    if (selected) cy.getElementById(selected).select();
  }, [selected, graphKey, focus, layout, theme]);
  const changeZoom = (factor: number) => {
    const cy = core.current;
    if (cy)
      cy.zoom({
        level: Math.min(3, Math.max(0.08, cy.zoom() * factor)),
        renderedPosition: { x: cy.width() / 2, y: cy.height() / 2 },
      });
  };
  return (
    <div className="graph-canvas-shell">
      <div
        className="graph-canvas"
        ref={host}
        tabIndex={0}
        role="group"
        aria-label="문서 연결 그래프"
        aria-describedby="graph-keyboard-help"
        onKeyDown={(event) => {
          const cy = core.current;
          if (!cy) return;
          if (event.key === "+" || event.key === "=") {
            event.preventDefault();
            changeZoom(1.2);
          } else if (event.key === "-") {
            event.preventDefault();
            changeZoom(1 / 1.2);
          } else if (
            ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"].includes(
              event.key,
            )
          ) {
            event.preventDefault();
            cy.panBy({
              x:
                event.key === "ArrowLeft"
                  ? 40
                  : event.key === "ArrowRight"
                    ? -40
                    : 0,
              y:
                event.key === "ArrowUp"
                  ? 40
                  : event.key === "ArrowDown"
                    ? -40
                    : 0,
            });
          } else if (event.key === "Enter" && selected) {
            event.preventDefault();
            callbacks.current.open(selected);
          }
        }}
      />
      <div className="graph-zoom-controls">
        <Button
          variant="secondary"
          aria-label="그래프 축소"
          onClick={() => changeZoom(1 / 1.2)}
        >
          <Minus size={17} />
        </Button>
        <span>{zoom}%</span>
        <Button
          variant="secondary"
          aria-label="그래프 확대"
          onClick={() => changeZoom(1.2)}
        >
          <Plus size={17} />
        </Button>
        <Button
          variant="secondary"
          aria-label="그래프 전체 맞춤"
          onClick={() => {
            if (core.current) fitGraph(core.current);
          }}
        >
          <Maximize size={17} />
        </Button>
      </div>
      <p className="graph-canvas-help" id="graph-keyboard-help">
        드래그로 이동 · 휠/두 손가락으로 확대 · 방향키와 +/− 사용 · 아래 문서
        목록으로도 탐색할 수 있습니다.
      </p>
    </div>
  );
}
