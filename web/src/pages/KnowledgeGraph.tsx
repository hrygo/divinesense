import { Loader2 } from "lucide-react";
import { lazy, Suspense } from "react";

const KnowledgeGraphView = lazy(() => import("@/components/KnowledgeGraphView"));

const KnowledgeGraph = () => {
  return (
    <div className="w-full h-full overflow-hidden">
      <Suspense
        fallback={
          <div className="flex items-center justify-center h-full">
            <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
          </div>
        }
      >
        <KnowledgeGraphView className="w-full h-[calc(100vh-120px)] border rounded-lg overflow-hidden bg-background shadow-sm" />
      </Suspense>
    </div>
  );
};

export default KnowledgeGraph;
