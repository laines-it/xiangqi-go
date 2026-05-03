const BOARD_COLS = 9;
const BOARD_ROWS = 10;
const CELL_SIZE = 57;
const PADDING = 30;
const PIECE_RADIUS = 23;

const NO_PIECE = 0;
const W_ROOK = 1;
const W_ADVISOR = 2;
const W_CANNON = 3;
const W_PAWN = 4;
const W_KNIGHT = 5;
const W_BISHOP = 6;
const W_KING = 7;
const B_ROOK = 9;
const B_ADVISOR = 10;
const B_CANNON = 11;
const B_PAWN = 12;
const B_KNIGHT = 13;
const B_BISHOP = 14;
const B_KING = 15;

const PIECE_CHARS = {
  [W_ROOK]: "\u4FE5",
  [W_ADVISOR]: "\u4ED5",
  [W_CANNON]: "\u70AE",
  [W_PAWN]: "\u5175",
  [W_KNIGHT]: "\u508C",
  [W_BISHOP]: "\u76F8",
  [W_KING]: "\u5E25",
  [B_ROOK]: "\u8ECA",
  [B_ADVISOR]: "\u58EB",
  [B_CANNON]: "\u7832",
  [B_PAWN]: "\u5352",
  [B_KNIGHT]: "\u99AC",
  [B_BISHOP]: "\u8C61",
  [B_KING]: "\u5C07",
};

function pieceColor(pc) {
  if (pc >= 1 && pc <= 7) return "red";
  if (pc >= 9 && pc <= 15) return "black";
  return null;
}

class BoardRenderer {
  constructor(canvas) {
    this.canvas = canvas;
    this.ctx = canvas.getContext("2d");
    this.flipped = false;
    this.selectedSq = -1;
    this.hoverSq = -1;
    this.legalTargets = [];
    this.lastFrom = -1;
    this.lastTo = -1;
    this.boardData = new Array(90).fill(0);

    this.canvas.width = PADDING * 2 + (BOARD_COLS - 1) * CELL_SIZE;
    this.canvas.height = PADDING * 2 + (BOARD_ROWS - 1) * CELL_SIZE;
  }

  sqToCanvas(sq) {
    const file = sq % 9;
    const rank = Math.floor(sq / 9);
    if (this.flipped) {
      return {
        x: PADDING + (8 - file) * CELL_SIZE,
        y: PADDING + rank * CELL_SIZE,
      };
    }
    return {
      x: PADDING + file * CELL_SIZE,
      y: PADDING + (9 - rank) * CELL_SIZE,
    };
  }

  canvasToSq(px, py) {
    let bestDist = Infinity;
    let bestSq = -1;
    for (let sq = 0; sq < 90; sq += 1) {
      const { x, y } = this.sqToCanvas(sq);
      const dist = Math.hypot(px - x, py - y);
      if (dist < PIECE_RADIUS + 7 && dist < bestDist) {
        bestDist = dist;
        bestSq = sq;
      }
    }
    return bestSq;
  }

  setHover(sq) {
    if (sq === this.hoverSq) return;
    this.hoverSq = sq;
    this.draw();
  }

  draw() {
    const ctx = this.ctx;
    const width = this.canvas.width;
    const height = this.canvas.height;

    ctx.clearRect(0, 0, width, height);
    this.drawBoardSurface(ctx, width, height);
    this.drawGrid(ctx);
    this.drawPalaceDiagonals(ctx);
    this.drawRiver(ctx);

    if (this.lastFrom >= 0 && this.lastTo >= 0) {
      this.drawSquareHighlight(ctx, this.lastFrom, "rgba(214, 62, 62, 0.20)");
      this.drawSquareHighlight(ctx, this.lastTo, "rgba(214, 62, 62, 0.28)");
    }
    if (this.selectedSq >= 0) {
      this.drawSquareHighlight(ctx, this.selectedSq, "rgba(24, 118, 108, 0.28)");
    }
    if (this.hoverSq >= 0 && this.boardData[this.hoverSq] !== NO_PIECE) {
      this.drawSquareHighlight(ctx, this.hoverSq, "rgba(10, 32, 44, 0.08)");
    }

    this.drawLegalTargets(ctx);

    for (let sq = 0; sq < 90; sq += 1) {
      const pc = this.boardData[sq];
      if (pc !== NO_PIECE) this.drawPiece(ctx, sq, pc);
    }
  }

  drawBoardSurface(ctx, width, height) {
    const gradient = ctx.createLinearGradient(0, 0, width, height);
    gradient.addColorStop(0, "#efd28b");
    gradient.addColorStop(0.55, "#e3ba67");
    gradient.addColorStop(1, "#dcae58");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, width, height);

    ctx.strokeStyle = "rgba(72, 45, 20, 0.26)";
    ctx.lineWidth = 2;
    ctx.strokeRect(8, 8, width - 16, height - 16);
  }

  drawGrid(ctx) {
    ctx.strokeStyle = "#4a301b";
    ctx.lineWidth = 1.2;

    for (let r = 0; r < BOARD_ROWS; r += 1) {
      const y = PADDING + r * CELL_SIZE;
      ctx.beginPath();
      ctx.moveTo(PADDING, y);
      ctx.lineTo(PADDING + (BOARD_COLS - 1) * CELL_SIZE, y);
      ctx.stroke();
    }

    for (let c = 0; c < BOARD_COLS; c += 1) {
      const x = PADDING + c * CELL_SIZE;
      if (c === 0 || c === BOARD_COLS - 1) {
        ctx.beginPath();
        ctx.moveTo(x, PADDING);
        ctx.lineTo(x, PADDING + (BOARD_ROWS - 1) * CELL_SIZE);
        ctx.stroke();
      } else {
        ctx.beginPath();
        ctx.moveTo(x, PADDING);
        ctx.lineTo(x, PADDING + 4 * CELL_SIZE);
        ctx.stroke();
        ctx.beginPath();
        ctx.moveTo(x, PADDING + 5 * CELL_SIZE);
        ctx.lineTo(x, PADDING + 9 * CELL_SIZE);
        ctx.stroke();
      }
    }
  }

  drawPalaceDiagonals(ctx) {
    const palaces = [
      { rank0: 0, rank2: 2 },
      { rank0: 7, rank2: 9 },
    ];
    ctx.strokeStyle = "#4a301b";
    ctx.lineWidth = 1.2;

    for (const palace of palaces) {
      const tl = this.sqToCanvas(3 + palace.rank0 * 9);
      const tr = this.sqToCanvas(5 + palace.rank0 * 9);
      const bl = this.sqToCanvas(3 + palace.rank2 * 9);
      const br = this.sqToCanvas(5 + palace.rank2 * 9);
      ctx.beginPath();
      ctx.moveTo(tl.x, tl.y);
      ctx.lineTo(br.x, br.y);
      ctx.stroke();
      ctx.beginPath();
      ctx.moveTo(tr.x, tr.y);
      ctx.lineTo(bl.x, bl.y);
      ctx.stroke();
    }
  }

  drawRiver(ctx) {
    const y = PADDING + 4.5 * CELL_SIZE;
    ctx.save();
    ctx.font = "600 23px Georgia, 'Times New Roman', serif";
    ctx.fillStyle = "rgba(74, 48, 27, 0.70)";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText("\u695A \u6CB3", PADDING + 2 * CELL_SIZE, y);
    ctx.fillText("\u6C49 \u754C", PADDING + 6 * CELL_SIZE, y);
    ctx.restore();
  }

  drawSquareHighlight(ctx, sq, color) {
    const { x, y } = this.sqToCanvas(sq);
    ctx.fillStyle = color;
    ctx.fillRect(x - CELL_SIZE / 2, y - CELL_SIZE / 2, CELL_SIZE, CELL_SIZE);
  }

  drawLegalTargets(ctx) {
    for (const target of this.legalTargets) {
      const { x, y } = this.sqToCanvas(target);
      ctx.beginPath();
      if (this.boardData[target] !== NO_PIECE) {
        ctx.arc(x, y, PIECE_RADIUS - 1, 0, Math.PI * 2);
        ctx.strokeStyle = "rgba(24, 118, 108, 0.78)";
        ctx.lineWidth = 3;
        ctx.stroke();
      } else {
        ctx.arc(x, y, 6, 0, Math.PI * 2);
        ctx.fillStyle = "rgba(24, 118, 108, 0.78)";
        ctx.fill();
      }
    }
  }

  drawPiece(ctx, sq, pc) {
    const { x, y } = this.sqToCanvas(sq);
    const isRed = pieceColor(pc) === "red";
    const stroke = isRed ? "#c62f2f" : "#17232f";

    ctx.save();
    ctx.shadowColor = "rgba(21, 26, 36, 0.24)";
    ctx.shadowBlur = 8;
    ctx.shadowOffsetY = 3;
    ctx.beginPath();
    ctx.arc(x, y, PIECE_RADIUS, 0, Math.PI * 2);
    ctx.fillStyle = "#fff4dd";
    ctx.fill();
    ctx.restore();

    ctx.beginPath();
    ctx.arc(x, y, PIECE_RADIUS, 0, Math.PI * 2);
    ctx.lineWidth = 2.2;
    ctx.strokeStyle = stroke;
    ctx.stroke();

    ctx.beginPath();
    ctx.arc(x, y, PIECE_RADIUS - 5, 0, Math.PI * 2);
    ctx.lineWidth = 1.2;
    ctx.strokeStyle = stroke;
    ctx.stroke();

    ctx.font = "700 22px KaiTi, STKaiti, SimSun, serif";
    ctx.fillStyle = stroke;
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(PIECE_CHARS[pc] || "?", x, y + 1);
  }

  update(boardArr, lastFrom, lastTo) {
    this.boardData = boardArr;
    this.lastFrom = lastFrom;
    this.lastTo = lastTo;
    this.draw();
  }

  setSelected(sq, targets) {
    this.selectedSq = sq;
    this.legalTargets = targets || [];
    this.draw();
  }

  clearSelection() {
    this.selectedSq = -1;
    this.legalTargets = [];
    this.draw();
  }
}
