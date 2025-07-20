import { GameGrid } from './components/GameGrid';
import { SolutionsSidebar } from './components/SolutionsSidebar';
import { useGameState } from './hooks/useGameState';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';

function App() {
  const gameLogic = useGameState();

  return (
    <div className="min-h-screen bg-gradient-to-br from-background to-muted p-4">
      <div className="mx-auto">
        {/* Header */}
        <Card className="mb-6 bg-card/80 backdrop-blur-sm shadow-lg">
          <CardHeader>
            <CardTitle className="text-2xl font-bold text-center">
              5×5 Word Square
            </CardTitle>
            <CardDescription className="text-center">
              Create words that read the same horizontally and vertically!
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Alert>
              <AlertDescription>
                <div className="space-y-2">
                  <p className="font-medium">How to play:</p>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-sm">
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="bg-blue-50">Click</Badge>
                      <span>Select row (blue) or column (green)</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">Type</Badge>
                      <span>Fill cells with letters</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">Tab</Badge>
                      <span>Toggle direction</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">Arrows</Badge>
                      <span>Navigate between cells</span>
                    </div>
                  </div>
                </div>
              </AlertDescription>
            </Alert>
          </CardContent>
        </Card>

        {/* Game Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Game Grid */}
          <div className="lg:col-span-2 flex justify-center items-start">
            <GameGrid
              gameState={gameLogic.gameState}
              selectedCell={gameLogic.selectedCell}
              selectedDirection={gameLogic.selectedDirection}
              onCellClick={gameLogic.handleCellClick}
              onKeyDown={gameLogic.handleKeyDown}
            />
          </div>

          {/* Solutions Sidebar */}
          <div className="lg:col-span-1">
            <SolutionsSidebar gameState={gameLogic.gameState} />
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;