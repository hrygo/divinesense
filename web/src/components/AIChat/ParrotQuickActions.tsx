import type { ParrotAgentI18n } from "@/hooks/useParrots";
import { useAvailableParrots } from "@/hooks/useParrots";
import { cn } from "@/lib/utils";
import { PARROT_ICONS } from "@/types/parrot";

interface ParrotQuickActionsProps {
  currentParrot: ParrotAgentI18n | null;
  onParrotChange: (parrot: ParrotAgentI18n) => void;
  disabled?: boolean;
}

export function ParrotQuickActions({ currentParrot, onParrotChange, disabled = false }: ParrotQuickActionsProps) {
  const availableParrots = useAvailableParrots();

  const handleParrotSelect = (parrot: ParrotAgentI18n) => {
    if (!disabled) {
      onParrotChange(parrot);
    }
  };

  return (
    <div className="flex items-center space-x-2 w-full overflow-x-auto pb-2">
      {availableParrots.map((parrot) => {
        const isSelected = currentParrot?.id === parrot.id;
        const colorClass = getColorClass(parrot.color);
        const icon = PARROT_ICONS[parrot.id] || parrot.icon;

        return (
          <button
            key={parrot.id}
            onClick={() => handleParrotSelect(parrot)}
            disabled={disabled}
            aria-label={`Switch to ${parrot.displayName}`}
            aria-pressed={isSelected}
            className={cn(
              "flex-shrink-0 flex items-center space-x-2 px-4 py-3 rounded-lg border-2 transition-all",
              "hover:shadow-md disabled:opacity-50 disabled:cursor-not-allowed",
              "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2",
              isSelected
                ? `${colorClass} border-current shadow-sm`
                : "border-zinc-200 dark:border-zinc-700 hover:border-zinc-300 dark:hover:border-zinc-600 bg-white dark:bg-zinc-800",
            )}
          >
            {icon.startsWith("/") ? (
              <img src={icon} alt={parrot.displayName} className="w-7 h-7 object-contain" />
            ) : (
              <span className="text-2xl" role="img" aria-label={parrot.displayName}>
                {icon}
              </span>
            )}
            <div className="text-left min-w-0">
              <div className="flex items-baseline gap-1.5 overflow-hidden">
                <div className={cn("font-semibold text-sm truncate", isSelected ? "text-current" : "text-zinc-900 dark:text-zinc-100")}>
                  {parrot.displayName}
                </div>
                <div
                  className={cn(
                    "text-[11px] opacity-60 truncate font-medium",
                    isSelected ? "text-current" : "text-zinc-400 dark:text-zinc-500",
                  )}
                >
                  {parrot.displayNameAlt}
                </div>
              </div>
              <div
                className={cn(
                  "text-[11px] line-clamp-1 mt-0.5 max-w-[120px]",
                  isSelected ? "text-current opacity-90" : "text-zinc-500 dark:text-zinc-400",
                )}
              >
                {parrot.description}
              </div>
            </div>
            {isSelected && (
              <div className="ml-auto">
                <div className="w-2 h-2 rounded-full bg-current" />
              </div>
            )}
          </button>
        );
      })}
    </div>
  );
}

function getColorClass(color: string): string {
  const colorMap: Record<string, string> = {
    blue: "bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 border-blue-300 dark:border-blue-700",
    indigo: "bg-indigo-50 dark:bg-indigo-900/20 text-indigo-700 dark:text-indigo-300 border-indigo-300 dark:border-indigo-700",
    purple: "bg-purple-50 dark:bg-purple-900/20 text-purple-700 dark:text-purple-300 border-purple-300 dark:border-purple-700",
    orange: "bg-orange-50 dark:bg-orange-900/20 text-orange-700 dark:text-orange-300 border-orange-300 dark:border-orange-700",
    pink: "bg-pink-50 dark:bg-pink-900/20 text-pink-700 dark:text-pink-300 border-pink-300 dark:border-pink-700",
    gray: "bg-gray-50 dark:bg-gray-900/20 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-700",
    green: "bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300 border-green-300 dark:border-green-700",
    red: "bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300 border-red-300 dark:border-red-700",
    yellow: "bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-300 border-yellow-300 dark:border-yellow-700",
  };

  return colorMap[color] || colorMap.gray;
}
