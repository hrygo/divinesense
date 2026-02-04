import {
  ArchiveIcon,
  CheckIcon,
  ChevronRightIcon,
  GlobeIcon,
  LogOutIcon,
  PaletteIcon,
  SettingsIcon,
  SquareUserIcon,
  User2Icon,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/contexts/AuthContext";
import useCurrentUser from "@/hooks/useCurrentUser";
import useIsMobile from "@/hooks/useIsMobile";
import useNavigateTo from "@/hooks/useNavigateTo";
import { useUpdateUserGeneralSetting } from "@/hooks/useUserQueries";
import { cn } from "@/lib/utils";
import { Routes } from "@/router";
import { loadLocale } from "@/utils/i18n";
import { getThemeWithFallback, loadTheme, THEME_OPTIONS } from "@/utils/theme";
import UserAvatar from "./UserAvatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "./ui/sheet";

interface Props {
  collapsed?: boolean;
}

const UserMenu = (props: Props) => {
  const { collapsed } = props;
  const isMobile = useIsMobile();
  const { t, i18n } = useTranslation();
  const navigateTo = useNavigateTo();
  const currentUser = useCurrentUser();
  const { userGeneralSetting, refetchSettings, logout } = useAuth();
  const { mutate: updateUserGeneralSetting } = useUpdateUserGeneralSetting(currentUser?.name);
  const currentLocale = i18n.language;
  const currentTheme = getThemeWithFallback(userGeneralSetting?.theme);
  const [showUserMenuSheet, setShowUserMenuSheet] = useState(false);
  const [showThemeSheet, setShowThemeSheet] = useState(false);

  const handleLocaleChange = async () => {
    const nextLocale = currentLocale === "en" ? "zh-Hans" : "en";
    if (!currentUser) return;
    // Apply locale immediately for instant UI feedback and persist to localStorage
    loadLocale(nextLocale);
    // Persist to user settings
    updateUserGeneralSetting(
      { generalSetting: { locale: nextLocale }, updateMask: ["locale"] },
      {
        onSuccess: () => {
          refetchSettings();
        },
      },
    );
  };

  const handleThemeChange = async (theme: string) => {
    loadTheme(theme);
    updateUserGeneralSetting(
      { generalSetting: { theme }, updateMask: ["theme"] },
      {
        onSuccess: () => {
          refetchSettings();
        },
      },
    );
    setShowThemeSheet(false);
    // Return to main menu after theme selection
    setTimeout(() => setShowUserMenuSheet(true), 100);
  };

  const handleSignOut = async () => {
    // First, clear auth state and cache BEFORE doing anything else
    await logout();

    try {
      // Then clear user-specific localStorage items
      // Preserve app-wide settings (theme, locale, view preferences, tag view settings)
      const keysToPreserve = ["memos-theme", "memos-locale", "memos-view-setting", "tag-view-as-tree", "tag-tree-auto-expand"];
      const keysToRemove: string[] = [];

      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && !keysToPreserve.includes(key)) {
          keysToRemove.push(key);
        }
      }

      keysToRemove.forEach((key) => localStorage.removeItem(key));
    } catch {
      // Ignore errors from localStorage operations
    }

    // Always redirect to auth page (use replace to prevent back navigation)
    window.location.replace(Routes.AUTH);
  };

  return (
    <>
      {/* Desktop: DropdownMenu */}
      {!isMobile && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild disabled={!currentUser}>
            <div
              className={cn(
                "min-h-[44px] w-auto flex flex-row justify-start items-center cursor-pointer text-foreground",
                collapsed ? "px-1" : "px-3",
              )}
            >
              {currentUser?.avatarUrl ? (
                <UserAvatar className="shrink-0" avatarUrl={currentUser?.avatarUrl} />
              ) : (
                <User2Icon className="w-6 mx-auto h-auto text-muted-foreground" />
              )}
              {!collapsed && (
                <span className="ml-2 text-lg font-medium text-foreground grow truncate">
                  {currentUser?.displayName || currentUser?.username}
                </span>
              )}
            </div>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuItem onClick={() => navigateTo(`/u/${encodeURIComponent(currentUser?.username ?? "")}`)}>
              <SquareUserIcon className="size-4 text-muted-foreground" />
              {t("common.profile")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => navigateTo(Routes.ARCHIVED)}>
              <ArchiveIcon className="size-4 text-muted-foreground" />
              {t("common.archived")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={handleLocaleChange}>
              <GlobeIcon className="size-4 text-muted-foreground bg-transparent" />
              <span className="text-sm">{currentLocale === "en" ? "中文" : "English"}</span>
            </DropdownMenuItem>
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <PaletteIcon className="size-4 text-muted-foreground" />
                {t("setting.preference-section.theme")}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                {THEME_OPTIONS.map((option) => (
                  <DropdownMenuItem key={option.value} onClick={() => handleThemeChange(option.value)}>
                    {currentTheme === option.value && <CheckIcon className="w-4 h-auto" />}
                    {currentTheme !== option.value && <span className="w-4" />}
                    {option.label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuSubContent>
            </DropdownMenuSub>
            <DropdownMenuItem onClick={() => navigateTo(Routes.SETTING)}>
              <SettingsIcon className="size-4 text-muted-foreground" />
              {t("common.settings")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={handleSignOut}>
              <LogOutIcon className="size-4 text-muted-foreground" />
              {t("common.sign-out")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}

      {/* Mobile: Avatar button that opens Sheet */}
      {isMobile && (
        <>
          <button
            onClick={() => setShowUserMenuSheet(true)}
            className={cn(
              "min-h-[44px] w-auto flex flex-row justify-start items-center cursor-pointer text-foreground bg-transparent",
              collapsed ? "px-1" : "px-3",
            )}
          >
            {currentUser?.avatarUrl ? (
              <UserAvatar className="shrink-0" avatarUrl={currentUser?.avatarUrl} />
            ) : (
              <User2Icon className="w-6 mx-auto h-auto text-muted-foreground" />
            )}
            {!collapsed && (
              <span className="ml-2 text-lg font-medium text-foreground grow truncate">
                {currentUser?.displayName || currentUser?.username}
              </span>
            )}
          </button>

          {/* Mobile Action Sheet */}
          <Sheet open={showUserMenuSheet} onOpenChange={setShowUserMenuSheet}>
            <SheetContent side="bottom" className="px-4 pb-6">
              <SheetHeader>
                <SheetTitle>{currentUser?.displayName || currentUser?.username}</SheetTitle>
              </SheetHeader>
              <div className="mt-4">
                <button
                  onClick={() => {
                    navigateTo(`/u/${encodeURIComponent(currentUser?.username ?? "")}`);
                    setShowUserMenuSheet(false);
                  }}
                  className="w-full flex items-center gap-3 px-4 py-3 border-b border-border last:border-0 active:bg-muted/50"
                >
                  <SquareUserIcon className="w-5 h-5 text-muted-foreground shrink-0" />
                  <span className="text-sm">{t("common.profile")}</span>
                </button>
                <button
                  onClick={() => {
                    navigateTo(Routes.ARCHIVED);
                    setShowUserMenuSheet(false);
                  }}
                  className="w-full flex items-center gap-3 px-4 py-3 border-b border-border last:border-0 active:bg-muted/50"
                >
                  <ArchiveIcon className="w-5 h-5 text-muted-foreground shrink-0" />
                  <span className="text-sm">{t("common.archived")}</span>
                </button>
                <button
                  onClick={() => {
                    handleLocaleChange();
                    setShowUserMenuSheet(false);
                  }}
                  className="w-full flex items-center gap-3 px-4 py-3 border-b border-border last:border-0 active:bg-muted/50"
                >
                  <GlobeIcon className="w-5 h-5 text-muted-foreground shrink-0" />
                  <span className="text-sm">{currentLocale === "en" ? "中文" : "English"}</span>
                </button>
                <button
                  onClick={() => {
                    setShowUserMenuSheet(false);
                    setTimeout(() => setShowThemeSheet(true), 100);
                  }}
                  className="w-full flex items-center justify-between px-4 py-3 border-b border-border last:border-0 active:bg-muted/50"
                >
                  <div className="flex items-center gap-3">
                    <PaletteIcon className="w-5 h-5 text-muted-foreground shrink-0" />
                    <span className="text-sm">{t("setting.preference-section.theme")}</span>
                  </div>
                  <ChevronRightIcon className="w-5 h-5 text-muted-foreground" />
                </button>
                <button
                  onClick={() => {
                    navigateTo(Routes.SETTING);
                    setShowUserMenuSheet(false);
                  }}
                  className="w-full flex items-center gap-3 px-4 py-3 border-b border-border last:border-0 active:bg-muted/50"
                >
                  <SettingsIcon className="w-5 h-5 text-muted-foreground shrink-0" />
                  <span className="text-sm">{t("common.settings")}</span>
                </button>
                <button onClick={handleSignOut} className="w-full flex items-center gap-3 px-4 py-3 active:bg-muted/50 text-destructive">
                  <LogOutIcon className="w-5 h-5 text-muted-foreground shrink-0" />
                  <span className="text-sm">{t("common.sign-out")}</span>
                </button>
              </div>
            </SheetContent>
          </Sheet>

          {/* Theme Sub Sheet */}
          <Sheet open={showThemeSheet} onOpenChange={setShowThemeSheet}>
            <SheetContent side="bottom" className="px-4 pb-6">
              <SheetHeader>
                <SheetTitle>{t("setting.preference-section.theme")}</SheetTitle>
              </SheetHeader>
              <div className="mt-4">
                {THEME_OPTIONS.filter((opt) => ["system", "default", "default-dark"].includes(opt.value)).map((theme) => (
                  <button
                    key={theme.value}
                    onClick={() => handleThemeChange(theme.value)}
                    className="w-full flex items-center justify-between px-4 py-3 border-b border-border last:border-0 active:bg-muted/50"
                  >
                    <span className="text-sm">{theme.label}</span>
                    {currentTheme === theme.value && <CheckIcon className="w-5 h-5 text-green-600" />}
                  </button>
                ))}
              </div>
            </SheetContent>
          </Sheet>
        </>
      )}
    </>
  );
};

export default UserMenu;
