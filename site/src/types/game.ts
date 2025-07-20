export interface CellPosition {
  row: number;
  col: number;
}

export type Direction = 'row' | 'col';

export interface GameState {
  grid: string[][];
}

export interface WordSolution {
  pattern: string;
  words: string[];
}

export interface GameSolutions {
  rows: WordSolution[];
  cols: WordSolution[];
}