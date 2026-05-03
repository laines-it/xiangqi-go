let board;
let playerSide = 0;
let aiThinking = false;
let searchDepth = 4;
let threadCount = 2;
let selectedAlgorithm = "alpha-beta";
let interactionState = 0;
let moveLog = [];
let engineInfo = null;

const STATE_IDLE = 0;
const STATE_SELECTED = 1;

const REQUIRED_ENGINE_APIS = [
  "engineNewGame",
  "engineGetBoard",
  "engineGetLegalMovesFrom",
  "engineDoMoveBySquares",
  "engineUndoMove",
  "engineSearch",
  "engineGetEngineInfo",
];

const ALGORITHM_LABELS = {
  "alpha-beta": "Alpha-Beta",
  ybwc: "YBWC",
  lazysmp: "LazySMP",
  pikafish: "Pikafish",
};

function missingEngineApis() {
  return REQUIRED_ENGINE_APIS.filter((name) => typeof window[name] !== "function");
}

async function initWasm() {
  const go = new Go();
  const wasmUrl = new URL("godogpaw.wasm", document.baseURI);
  const response = await fetch(wasmUrl);
  if (!response.ok) {
    throw new Error(`Failed to fetch ${wasmUrl.pathname}: ${response.status}`);
  }

  let result;
  if (WebAssembly.instantiateStreaming) {
    try {
      result = await WebAssembly.instantiateStreaming(response.clone(), go.importObject);
    } catch (_) {
      const bytes = await response.arrayBuffer();
      result = await WebAssembly.instantiate(bytes, go.importObject);
    }
  } else {
    const bytes = await response.arrayBuffer();
    result = await WebAssembly.instantiate(bytes, go.importObject);
  }

  go.run(result.instance).catch((err) => {
    console.error("Go runtime error:", err);
    setStatus("Engine runtime stopped.", "error");
  });

  await new Promise((resolve) => setTimeout(resolve, 0));
  const missing = missingEngineApis();
  if (missing.length > 0) {
    throw new Error(`Engine API missing: ${missing.join(", ")}`);
  }
}

function setStatus(text, tone = "neutral") {
  document.getElementById("status-text").textContent = text;
  document.getElementById("status-bar").dataset.tone = tone;
}

function setThinking(on) {
  aiThinking = on;
  document.getElementById("app").classList.toggle("thinking", on);
  document.querySelectorAll("button, select, input").forEach((el) => {
    if (el.id !== "btn-flip") el.disabled = on;
  });
  if (!on) updateAlgorithmControls();
}

function parseEngineJSON(value, fallback) {
  if (typeof value !== "string") return value ?? fallback;
  try {
    return JSON.parse(value);
  } catch (err) {
    console.error("Bad engine JSON:", value, err);
    return fallback;
  }
}

function refreshBoard() {
  const state = parseEngineJSON(engineGetBoard(), null);
  if (!state) return true;

  board.update(state.board, state.lastMoveFrom, state.lastMoveTo);
  document.getElementById("stat-turn").textContent = sideName(state.sideToMove);
  document.getElementById("stat-last").textContent = state.lastMove || "-";

  if (state.isGameOver) {
    const playerLost = state.sideToMove === playerSide;
    setStatus(playerLost ? "Game over. You lose." : "Game over. You win.", playerLost ? "error" : "success");
    return true;
  }

  if (state.inCheck) {
    setStatus(state.sideToMove === playerSide ? "Check. Your turn." : "Check.", "warning");
  } else if (state.sideToMove === playerSide) {
    setStatus("Your turn.", "success");
  } else {
    setStatus("Engine turn.", "neutral");
  }

  return false;
}

async function aiMove() {
  setThinking(true);
  setStatus(`${algorithmLabel(selectedAlgorithm)} thinking...`, "loading");
  await new Promise((resolve) => setTimeout(resolve, 20));

  try {
    const raw = await engineSearch(selectedAlgorithm, searchDepth, threadCount);
    const result = parseEngineJSON(raw, { ok: false, message: "Search returned invalid data." });
    updateSearchStats(result);

    if (!result.ok) {
      setStatus(result.message || "Engine has no legal move.", "error");
      return;
    }

    moveLog.push({
      side: "Engine",
      move: result.move,
      algorithm: algorithmLabel(result.algorithm),
      fallback: result.fallback,
    });
    renderMoveList();
    board.clearSelection();
    refreshBoard();

    if (result.message) {
      setStatus(result.message, result.fallback ? "warning" : "neutral");
    }
  } catch (err) {
    console.error("AI search error:", err);
    setStatus("Search failed.", "error");
  } finally {
    setThinking(false);
  }
}

function startNewGame() {
  playerSide = document.getElementById("sel-side").value === "black" ? 1 : 0;
  board.flipped = playerSide === 1;
  engineNewGame("");
  moveLog = [];
  interactionState = STATE_IDLE;
  board.clearSelection();
  clearSearchStats();
  renderMoveList();
  const gameOver = refreshBoard();
  if (!gameOver && playerSide === 1) aiMove();
}

function undoMove() {
  if (aiThinking) return;
  const undoneA = engineUndoMove();
  const undoneB = engineUndoMove();
  if (!undoneA && !undoneB) return;
  moveLog = moveLog.slice(0, Math.max(0, moveLog.length - 2));
  renderMoveList();
  board.clearSelection();
  interactionState = STATE_IDLE;
  clearSearchStats();
  refreshBoard();
}

function handleBoardClick(event) {
  if (aiThinking) return;

  const sq = eventToSquare(event);
  if (sq < 0) {
    clearSelection();
    return;
  }

  const state = parseEngineJSON(engineGetBoard(), null);
  if (!state || state.sideToMove !== playerSide || state.isGameOver) return;

  const pc = state.board[sq];
  const pcSide = pc === 0 ? -1 : pc <= 7 ? 0 : 1;

  if (interactionState === STATE_IDLE) {
    if (pcSide === playerSide) selectPiece(sq);
    return;
  }

  if (interactionState === STATE_SELECTED) {
    if (pcSide === playerSide) {
      selectPiece(sq);
    } else if (board.legalTargets.includes(sq)) {
      const from = board.selectedSq;
      const result = parseEngineJSON(engineDoMoveBySquares(from, sq), { ok: false });
      if (!result.ok) {
        setStatus(result.message || "Illegal move.", "error");
        return;
      }

      moveLog.push({ side: "You", move: result.move });
      renderMoveList();
      clearSelection();
      const gameOver = refreshBoard();
      if (!gameOver) aiMove();
    } else {
      clearSelection();
    }
  }
}

function handleBoardPointerMove(event) {
  if (!board) return;
  board.setHover(eventToSquare(event));
}

function eventToSquare(event) {
  const rect = board.canvas.getBoundingClientRect();
  const scaleX = board.canvas.width / rect.width;
  const scaleY = board.canvas.height / rect.height;
  const px = (event.clientX - rect.left) * scaleX;
  const py = (event.clientY - rect.top) * scaleY;
  return board.canvasToSq(px, py);
}

function selectPiece(sq) {
  const targets = parseEngineJSON(engineGetLegalMovesFrom(sq), []);
  board.setSelected(sq, targets);
  interactionState = STATE_SELECTED;
}

function clearSelection() {
  board.clearSelection();
  interactionState = STATE_IDLE;
}

function initControls() {
  document.getElementById("btn-new-game").addEventListener("click", startNewGame);
  document.getElementById("btn-undo").addEventListener("click", undoMove);
  document.getElementById("btn-flip").addEventListener("click", () => {
    board.flipped = !board.flipped;
    board.draw();
  });
  document.getElementById("sel-side").addEventListener("change", startNewGame);

  document.querySelectorAll("[data-algorithm]").forEach((btn) => {
    btn.addEventListener("click", () => {
      selectedAlgorithm = btn.dataset.algorithm;
      updateAlgorithmControls();
    });
  });

  document.querySelectorAll(".preset-btn").forEach((btn) => {
    btn.addEventListener("click", () => {
      searchDepth = Number(btn.dataset.depth);
      document.getElementById("slider-depth").value = String(searchDepth);
      document.querySelectorAll(".preset-btn").forEach((item) => item.classList.remove("active"));
      btn.classList.add("active");
      updateSearchLabels();
    });
  });

  document.getElementById("slider-depth").addEventListener("input", (event) => {
    searchDepth = Number(event.target.value);
    document.querySelectorAll(".preset-btn").forEach((btn) => {
      btn.classList.toggle("active", Number(btn.dataset.depth) === searchDepth);
    });
    updateSearchLabels();
  });

  document.getElementById("slider-threads").addEventListener("input", (event) => {
    threadCount = Number(event.target.value);
    updateSearchLabels();
  });

  updateAlgorithmControls();
  updateSearchLabels();
}

function updateAlgorithmControls() {
  document.querySelectorAll("[data-algorithm]").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.algorithm === selectedAlgorithm);
  });

  const threads = document.getElementById("slider-threads");
  threads.disabled = aiThinking || selectedAlgorithm !== "lazysmp";

  const info = engineInfo?.algorithms?.find((item) => item.id === selectedAlgorithm);
  const note = document.getElementById("algorithm-note");
  note.textContent = info?.note || "";
  note.classList.toggle("is-visible", Boolean(info?.note));
}

function updateSearchLabels() {
  document.getElementById("val-depth").textContent = String(searchDepth);
  document.getElementById("val-threads").textContent = String(threadCount);
}

function updateSearchStats(result) {
  document.getElementById("stat-score").textContent = result.ok ? formatScore(result.score) : "-";
  document.getElementById("stat-time").textContent = result.durationMs ? `${result.durationMs.toFixed(0)} ms` : "-";
  document.getElementById("stat-pv").textContent = result.pv?.length ? result.pv.join(" ") : "-";
}

function clearSearchStats() {
  document.getElementById("stat-score").textContent = "-";
  document.getElementById("stat-time").textContent = "-";
  document.getElementById("stat-pv").textContent = "-";
}

function renderMoveList() {
  const list = document.getElementById("move-list");
  list.textContent = "";
  for (const item of moveLog) {
    const li = document.createElement("li");
    const move = document.createElement("span");
    move.textContent = `${item.side}: ${item.move}`;
    li.append(move);
    if (item.algorithm) {
      const tag = document.createElement("small");
      tag.textContent = item.fallback ? `${item.algorithm} fallback` : item.algorithm;
      li.append(tag);
    }
    list.append(li);
  }
}

function sideName(side) {
  return side === 0 ? "Red" : "Black";
}

function algorithmLabel(id) {
  return ALGORITHM_LABELS[id] || id;
}

function formatScore(score) {
  if (Math.abs(score) > 30000) return score > 0 ? "Mate+" : "Mate-";
  const sign = score > 0 ? "+" : "";
  return `${sign}${score}`;
}

(async function main() {
  const canvas = document.getElementById("board-canvas");
  board = new BoardRenderer(canvas);
  board.draw();

  setStatus("Loading engine...", "loading");
  try {
    await initWasm();
    engineInfo = parseEngineJSON(engineGetEngineInfo(), null);
    if (engineInfo) {
      document.getElementById("runtime-badge").textContent =
        `${engineInfo.goos}/${engineInfo.goarch} P${engineInfo.maxProcs}`;
    }
  } catch (err) {
    console.error("Engine init error:", err);
    setStatus("Engine failed to load.", "error");
    return;
  }

  canvas.addEventListener("click", handleBoardClick);
  canvas.addEventListener("pointermove", handleBoardPointerMove);
  canvas.addEventListener("pointerleave", () => board.setHover(-1));
  initControls();
  startNewGame();
})();
