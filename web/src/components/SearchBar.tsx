/**
 * SearchBar - Intelligent Search Component
 *
 * Design Philosophy: "Subtle Sophistication"
 * - Clean, minimal interface with thoughtful micro-interactions
 * - Smooth mode transitions between AI and keyword search
 * - Elegant dropdown with glassmorphism effects
 * - Consistent spacing and visual hierarchy
 */

import { AlertCircle, FileText, Loader2, Search, Sparkles, X } from "lucide-react";
import { memo, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { useMemoFilterContext } from "@/contexts/MemoFilterContext";
import { useSemanticSearch } from "@/hooks/useAIQueries";
import useNavigateTo from "@/hooks/useNavigateTo";
import { cn } from "@/lib/utils";

interface SearchBarProps {
  className?: string;
}

const SearchBar = memo(function SearchBar({ className }: SearchBarProps) {
  const { t } = useTranslation();
  const { addFilter } = useMemoFilterContext();
  const [queryText, setQueryText] = useState("");
  const [isSemantic, setIsSemantic] = useState(true);
  const [isFocused, setIsFocused] = useState(false);
  const [dropdownPosition, setDropdownPosition] = useState({ top: 0, left: 0, width: 0 });
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const navigateTo = useNavigateTo();

  const {
    data: semanticResults,
    isLoading,
    isError,
  } = useSemanticSearch(queryText, {
    enabled: isSemantic && queryText.length > 1,
  });

  // Calculate dropdown position
  useEffect(() => {
    if (containerRef.current && (isFocused || queryText.length > 1)) {
      const rect = containerRef.current.getBoundingClientRect();
      setDropdownPosition({
        top: rect.bottom + 8,
        left: rect.left,
        width: rect.width,
      });
    }
  }, [isFocused, queryText]);

  const onTextChange = (event: React.ChangeEvent<HTMLInputElement>) => {
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
    if (e.key === "Escape") {
      setIsFocused(false);
      inputRef.current?.blur();
    }
  };

  const onMemoClick = (memoId: string) => {
    const id = memoId.split("/").pop();
    if (id) {
      navigateTo(`/m/${id}`);
      setQueryText("");
      setIsFocused(false);
    }
  };

  const handleClearSearch = () => {
    setQueryText("");
    inputRef.current?.focus();
  };

  const handleToggleMode = () => {
    setIsSemantic(!isSemantic);
    inputRef.current?.focus();
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsFocused(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const showDropdown = isSemantic && queryText.length > 1 && isFocused;
  const hasResults = semanticResults?.results && semanticResults.results.length > 0;

  return (
    <div className={cn("relative", className)} ref={containerRef}>
      {/* Search Input Container */}
      <div
        className={cn(
          "relative flex items-center gap-2 px-3 py-2.5 rounded-xl",
          "bg-card/50 backdrop-blur-sm",
          "border-2 transition-all duration-300 ease-out",
          isFocused ? "border-primary shadow-lg shadow-primary/5" : "border-border/50 hover:border-border",
        )}
      >
        {/* Mode Toggle Button */}
        <button
          onClick={handleToggleMode}
          className={cn(
            "p-1.5 rounded-lg transition-all duration-200",
            "flex items-center justify-center",
            isSemantic ? "bg-primary/10 text-primary hover:bg-primary/20" : "bg-muted/50 text-muted-foreground hover:bg-muted",
          )}
          aria-label={isSemantic ? t("search.switch-to-keyword") : t("search.switch-to-ai")}
          title={isSemantic ? t("search.ai-mode") : t("search.keyword-mode")}
        >
          {isSemantic ? <Sparkles className="w-4 h-4" /> : <Search className="w-4 h-4" />}
        </button>

        {/* Input Field */}
        <input
          ref={inputRef}
          type="text"
          value={queryText}
          onChange={onTextChange}
          onKeyDown={onKeyDown}
          onFocus={() => setIsFocused(true)}
          placeholder={isSemantic ? t("search.ai-placeholder") : t("memo.search_placeholder")}
          className={cn(
            "flex-1 bg-transparent border-0 outline-none text-foreground text-sm",
            "placeholder:text-muted-foreground/50",
            "transition-all duration-200",
          )}
        />

        {/* Clear Button */}
        {queryText && (
          <button
            onClick={handleClearSearch}
            className={cn(
              "p-1 rounded-lg transition-all duration-200",
              "text-muted-foreground hover:text-foreground hover:bg-muted",
              "active:scale-95",
            )}
            aria-label="Clear search"
          >
            <X className="w-4 h-4" />
          </button>
        )}

        {/* Focus Indicator Line */}
        <div
          className={cn(
            "absolute bottom-0 left-3 right-3 h-0.5 rounded-full",
            "transition-all duration-300 ease-out",
            "bg-gradient-to-r from-primary via-violet-500 to-primary",
            isFocused ? "opacity-100 scale-x-100" : "opacity-0 scale-x-0",
          )}
        />
      </div>

      {/* Dropdown Results - Portal */}
      {showDropdown &&
        createPortal(
          <div
            className="fixed z-50 animate-in fade-in slide-in-from-top-2 duration-200"
            style={{
              top: dropdownPosition.top,
              left: dropdownPosition.left,
              width: dropdownPosition.width,
            }}
          >
            <div className="bg-card/95 backdrop-blur-md border border-border/50 rounded-xl shadow-xl overflow-hidden">
              {/* Loading State */}
              {isLoading && (
                <div className="p-6 flex items-center justify-center text-muted-foreground">
                  <Loader2 className="w-5 h-5 animate-spin mr-3 text-primary" />
                  <span className="text-sm">{t("search.ai-searching")}</span>
                </div>
              )}

              {/* Error State */}
              {isError && (
                <div className="p-6">
                  <div className="flex items-center justify-center text-amber-600 dark:text-amber-400 mb-3">
                    <AlertCircle className="w-5 h-5 mr-2" />
                    <span className="text-sm font-medium">{t("search.ai-error")}</span>
                  </div>
                  <button
                    onClick={handleToggleMode}
                    className="w-full py-2 px-4 text-sm bg-primary/10 text-primary rounded-lg hover:bg-primary/20 transition-colors"
                  >
                    {t("search.fallback-to-keyword")}
                  </button>
                </div>
              )}

              {/* No Results State */}
              {!isLoading && !isError && !hasResults && (
                <div className="p-6 text-center">
                  <div className="w-12 h-12 mx-auto mb-3 rounded-full bg-muted/50 flex items-center justify-center">
                    <Search className="w-5 h-5 text-muted-foreground" />
                  </div>
                  <p className="text-sm text-muted-foreground">{t("search.no-results")}</p>
                </div>
              )}

              {/* Results List */}
              {!isLoading && !isError && hasResults && (
                <div className="max-h-80 overflow-y-auto">
                  {semanticResults.results.map((result, index) => (
                    <div
                      key={result.name}
                      className={cn(
                        "p-4 border-b border-border/50 last:border-0",
                        "hover:bg-muted/50 cursor-pointer group",
                        "transition-all duration-150",
                        "flex items-start gap-3",
                      )}
                      onClick={() => onMemoClick(result.name)}
                      style={{ animationDelay: `${index * 30}ms` }}
                    >
                      {/* Result Icon */}
                      <div className="mt-0.5 p-1.5 rounded-md bg-primary/10 group-hover:bg-primary/20 transition-colors">
                        <FileText className="w-4 h-4 text-primary" />
                      </div>

                      {/* Result Content */}
                      <div className="flex-1 min-w-0">
                        {/* Score Badge */}
                        <div className="flex items-center gap-2 mb-1.5">
                          <span
                            className={cn(
                              "inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium",
                              "bg-primary/10 text-primary",
                            )}
                          >
                            {Math.round(result.score * 100)}% match
                          </span>
                        </div>

                        {/* Snippet */}
                        <p className="text-sm text-foreground/80 line-clamp-2 leading-relaxed">{result.snippet}</p>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Footer Hint */}
              {!isLoading && !isError && hasResults && (
                <div className="px-4 py-2 bg-muted/30 border-t border-border/50">
                  <p className="text-xs text-muted-foreground text-center">
                    {t("search.navigate-hint") || "Press Enter to open • Esc to close"}
                  </p>
                </div>
              )}
            </div>
          </div>,
          document.body,
        )}
    </div>
  );
});

SearchBar.displayName = "SearchBar";

export default SearchBar;
