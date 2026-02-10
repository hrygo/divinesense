import { AlertCircleIcon, FileTextIcon, LoaderIcon, SearchIcon, SparklesIcon } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMemoFilterContext } from "@/contexts/MemoFilterContext";
import { useSemanticSearch } from "@/hooks/useAIQueries";
import useNavigateTo from "@/hooks/useNavigateTo";
import { cn } from "@/lib/utils";

const SearchBar = () => {
  const { t } = useTranslation();
  const { addFilter } = useMemoFilterContext();
  const [queryText, setQueryText] = useState("");
  const [isSemantic, setIsSemantic] = useState(true); // AI search as default
  const inputRef = useRef<HTMLInputElement>(null);
  const navigateTo = useNavigateTo();

  const {
    data: semanticResults,
    isLoading,
    isError,
  } = useSemanticSearch(queryText, {
    enabled: isSemantic && queryText.length > 1,
  });

  const onTextChange = (event: React.FormEvent<HTMLInputElement>) => {
    setQueryText(event.currentTarget.value);
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      if (!isSemantic) {
        const trimmedText = queryText.trim();
        if (trimmedText !== "") {
          const words = trimmedText.split(/\s+/);
          words.forEach((word) => {
            addFilter({
              factor: "contentSearch",
              value: word,
            });
          });
          setQueryText("");
        }
      }
    }
  };

  const onMemoClick = (memoId: string) => {
    const id = memoId.split("/").pop();
    if (id) {
      navigateTo(`/m/${id}`);
      setQueryText("");
    }
  };

  const handleToggleMode = () => {
    setIsSemantic(!isSemantic);
  };

  // AI Native Design: Icon in left side acts as mode toggle
  const ModeIcon = isSemantic ? SparklesIcon : SearchIcon;

  return (
    <div className="relative w-full h-auto flex flex-col z-20">
      <div
        className={cn(
          "relative w-full flex flex-row items-center rounded-lg transition-all duration-200",
          isSemantic && "bg-gradient-to-r from-blue-500/10 via-purple-500/10 to-blue-500/10 p-[1px]",
        )}
      >
        <div className="relative w-full flex flex-row items-center bg-sidebar rounded-lg">
          <button
            onClick={handleToggleMode}
            className={cn(
              "absolute left-2 p-0.5 rounded transition-colors z-10",
              isSemantic
                ? "text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300"
                : "text-sidebar-foreground/40 hover:text-sidebar-foreground/60",
            )}
            aria-label={isSemantic ? t("search.switch-to-keyword") : t("search.switch-to-ai")}
            title={isSemantic ? t("search.ai-mode") : t("search.keyword-mode")}
          >
            <ModeIcon className="w-4 h-4" />
          </button>
          <input
            className={cn(
              "w-full text-sidebar-foreground leading-6 bg-transparent border border-transparent text-sm rounded-lg p-1.5 pl-8 pr-3 outline-0",
              !isSemantic && "border-border",
            )}
            placeholder={isSemantic ? t("search.ai-placeholder") : t("memo.search-placeholder")}
            value={queryText}
            onChange={onTextChange}
            onKeyDown={onKeyDown}
            ref={inputRef}
          />
        </div>
      </div>

      {isSemantic && queryText.length > 1 && (
        <div className="absolute top-full mt-2 w-full bg-background border border-border rounded-lg shadow-lg overflow-hidden max-h-80 overflow-y-auto">
          {isLoading && (
            <div className="p-4 flex items-center justify-center text-muted-foreground">
              <LoaderIcon className="w-4 h-4 animate-spin mr-2" />
              {t("search.ai-searching")}
            </div>
          )}
          {isError && (
            <div className="p-4 text-center">
              <div className="flex items-center justify-center text-amber-600 dark:text-amber-400 mb-2">
                <AlertCircleIcon className="w-4 h-4 mr-2" />
                {t("search.ai-error")}
              </div>
              <button onClick={handleToggleMode} className="text-sm text-blue-600 dark:text-blue-400 hover:underline">
                {t("search.fallback-to-keyword")}
              </button>
            </div>
          )}
          {!isLoading && !isError && semanticResults?.results?.length === 0 && (
            <div className="p-4 text-center text-muted-foreground text-sm">{t("search.no-results")}</div>
          )}
          {!isLoading &&
            !isError &&
            semanticResults?.results?.map((result) => (
              <div
                key={result.name}
                className="p-3 border-b last:border-0 hover:bg-muted/50 cursor-pointer group"
                onClick={() => onMemoClick(result.name)}
              >
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center text-xs text-muted-foreground">
                    <FileTextIcon className="w-3 h-3 mr-1" />
                    {t("search.score")}: {(result.score * 100).toFixed(0)}%
                  </div>
                </div>
                <div className="text-sm line-clamp-2 text-foreground group-hover:text-blue-600 dark:group-hover:text-blue-400">
                  {result.snippet}
                </div>
              </div>
            ))}
        </div>
      )}
    </div>
  );
};

export default SearchBar;
