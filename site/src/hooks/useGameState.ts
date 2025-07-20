import { useState, useCallback } from 'react';
import { GameState, CellPosition, Direction } from '../types/game';

export function useGameState() {
  const [gameState, setGameState] = useState<GameState>({
    grid: Array(5).fill(null).map(() => Array(5).fill(''))
  });
  
  const [selectedCell, setSelectedCell] = useState<CellPosition | null>(null);
  const [selectedDirection, setSelectedDirection] = useState<Direction>('row');

  const handleCellClick = useCallback((position: CellPosition) => {
    if (selectedCell?.row === position.row && selectedCell?.col === position.col) {
      // Toggle direction if clicking the same cell
      setSelectedDirection(prev => prev === 'row' ? 'col' : 'row');
    } else {
      // Select new cell and default to row direction
      setSelectedCell(position);
      setSelectedDirection('row');
    }
  }, [selectedCell]);  const handleKeyDown = useCallback((position: CellPosition, event: React.KeyboardEvent) => {
    if (!selectedCell) return;

    switch (event.key) {
      case 'ArrowUp':
        event.preventDefault();
        setSelectedCell(prev => prev ? { ...prev, row: Math.max(0, prev.row - 1) } : null);
        break;
      case 'ArrowDown':
        event.preventDefault();
        setSelectedCell(prev => prev ? { ...prev, row: Math.min(4, prev.row + 1) } : null);
        break;
      case 'ArrowLeft':
        event.preventDefault();
        setSelectedCell(prev => prev ? { ...prev, col: Math.max(0, prev.col - 1) } : null);
        break;
      case 'ArrowRight':
        event.preventDefault();
        setSelectedCell(prev => prev ? { ...prev, col: Math.min(4, prev.col + 1) } : null);
        break;
      case 'Tab':
        event.preventDefault();
        setSelectedDirection(prev => prev === 'row' ? 'col' : 'row');
        break;
      case 'Backspace':
        event.preventDefault();
        if (gameState.grid[position.row][position.col] === '') {
          // Move to previous cell if current is empty
          const prevPosition = getPreviousPosition(selectedCell, selectedDirection);
          if (prevPosition) {
            setSelectedCell(prevPosition);
            setGameState(prev => ({
              ...prev,
              grid: prev.grid.map((row, rowIndex) =>
                row.map((cell, colIndex) =>
                  rowIndex === prevPosition.row && colIndex === prevPosition.col ? '' : cell
                )
              )
            }));
          }
        } else {
          // Clear current cell
          setGameState(prev => ({
            ...prev,
            grid: prev.grid.map((row, rowIndex) =>
              row.map((cell, colIndex) =>
                rowIndex === position.row && colIndex === position.col ? '' : cell
              )
            )
          }));
        }
        break;
      default:
        // Handle letter input
        if (event.key.length === 1 && /[a-zA-Z]/.test(event.key)) {
          event.preventDefault();
          const newValue = event.key.toUpperCase();
          
          // Update the cell value
          setGameState(prev => ({
            ...prev,
            grid: prev.grid.map((row, rowIndex) =>
              row.map((cell, colIndex) =>
                rowIndex === position.row && colIndex === position.col ? newValue : cell
              )
            )
          }));

          // Auto-advance to next cell
          const nextPosition = getNextPosition(selectedCell, selectedDirection);
          if (nextPosition) {
            setSelectedCell(nextPosition);
          }
        }
        break;
    }
  }, [selectedCell, selectedDirection, gameState.grid]);

  const getNextPosition = (current: CellPosition, direction: Direction): CellPosition | null => {
    if (direction === 'row') {
      return current.col < 4 ? { ...current, col: current.col + 1 } : null;
    } else {
      return current.row < 4 ? { ...current, row: current.row + 1 } : null;
    }
  };

  const getPreviousPosition = (current: CellPosition, direction: Direction): CellPosition | null => {
    if (direction === 'row') {
      return current.col > 0 ? { ...current, col: current.col - 1 } : null;
    } else {
      return current.row > 0 ? { ...current, row: current.row - 1 } : null;
    }
  };

  return {
    gameState,
    selectedCell,
    selectedDirection,
    handleCellClick,
    handleKeyDown,
  };
}