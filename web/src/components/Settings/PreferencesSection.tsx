import { create } from "@bufbuild/protobuf";
import { CheckIcon, ChevronRightIcon } from "lucide-react";
import { useState } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { useAuth } from "@/contexts/AuthContext";
import useIsMobile from "@/hooks/useIsMobile";
import { useUpdateUserGeneralSetting } from "@/hooks/useUserQueries";
import { cn } from "@/lib/utils";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { UserSetting_GeneralSetting, UserSetting_GeneralSettingSchema } from "@/types/proto/api/v1/user_service_pb";
import { loadLocale, useTranslate } from "@/utils/i18n";
import { convertVisibilityFromString, convertVisibilityToString } from "@/utils/memo";
import { loadTheme, THEME_OPTIONS } from "@/utils/theme";
import LocaleSelect from "../LocaleSelect";
import ThemeSelect from "../ThemeSelect";
import VisibilityIcon from "../VisibilityIcon";
import SettingGroup from "./SettingGroup";
import SettingRow from "./SettingRow";
import SettingSection from "./SettingSection";
import WebhookSection from "./WebhookSection";

const LOCALE_OPTIONS: Locale[] = ["en", "zh-Hans"];

const VISIBILITY_OPTIONS = [
  { value: Visibility.PRIVATE, labelKey: "memo.visibility.private" },
  { value: Visibility.PROTECTED, labelKey: "memo.visibility.protected" },
  { value: Visibility.PUBLIC, labelKey: "memo.visibility.public" },
];

const PreferencesSection = () => {
  const t = useTranslate();
  const { currentUser, userGeneralSetting: generalSetting, refetchSettings } = useAuth();
  const { mutate: updateUserGeneralSetting } = useUpdateUserGeneralSetting(currentUser?.name);
  const isMobile = useIsMobile();

  // Sheet states
  const [showLocaleSheet, setShowLocaleSheet] = useState(false);
  const [showThemeSheet, setShowThemeSheet] = useState(false);
  const [showVisibilitySheet, setShowVisibilitySheet] = useState(false);

  const handleLocaleSelectChange = async (locale: Locale) => {
    loadLocale(locale);
    updateUserGeneralSetting(
      { generalSetting: { locale }, updateMask: ["locale"] },
      {
        onSuccess: () => {
          refetchSettings();
        },
      },
    );
    setShowLocaleSheet(false);
  };

  const handleDefaultMemoVisibilityChanged = (value: string) => {
    updateUserGeneralSetting(
      { generalSetting: { memoVisibility: value }, updateMask: ["memo_visibility"] },
      {
        onSuccess: () => {
          refetchSettings();
        },
      },
    );
    setShowVisibilitySheet(false);
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
  };

  // Provide default values if setting is not loaded yet
  const setting: UserSetting_GeneralSetting =
    generalSetting ||
    create(UserSetting_GeneralSettingSchema, {
      locale: "en",
      memoVisibility: "PRIVATE",
      theme: "system",
    });

  // Mobile Action Sheet for Language
  const MobileLocaleSheet = () => (
    <Sheet open={showLocaleSheet} onOpenChange={setShowLocaleSheet}>
      <SheetContent side="bottom" className="px-4 pb-6">
        <SheetHeader>
          <SheetTitle>{t("common.language")}</SheetTitle>
        </SheetHeader>
        <div className="mt-4">
          {LOCALE_OPTIONS.map((locale) => (
            <button
              key={locale}
              onClick={() => handleLocaleSelectChange(locale)}
              className={cn(
                "w-full flex items-center justify-between px-4 py-3 border-b border-border last:border-0",
                "active:bg-muted/50",
                setting.locale === locale && "bg-muted/30",
              )}
            >
              <span className="text-sm">{locale === "en" ? "English" : locale === "zh-Hans" ? "简体中文" : "繁體中文"}</span>
              {setting.locale === locale && <CheckIcon className="w-5 h-5 text-green-600" />}
            </button>
          ))}
        </div>
      </SheetContent>
    </Sheet>
  );

  // Mobile Action Sheet for Theme
  const MobileThemeSheet = () => (
    <Sheet open={showThemeSheet} onOpenChange={setShowThemeSheet}>
      <SheetContent side="bottom" className="px-4 pb-6">
        <SheetHeader>
          <SheetTitle>{t("setting.preference-section.theme")}</SheetTitle>
        </SheetHeader>
        <div className="mt-4">
          {THEME_OPTIONS.filter((opt) => ["system", "light", "dark"].includes(opt.value)).map((theme) => (
            <button
              key={theme.value}
              onClick={() => handleThemeChange(theme.value)}
              className={cn(
                "w-full flex items-center justify-between px-4 py-3 border-b border-border last:border-0",
                "active:bg-muted/50",
                setting.theme === theme.value && "bg-muted/30",
              )}
            >
              <span className="text-sm">{theme.label}</span>
              {setting.theme === theme.value && <CheckIcon className="w-5 h-5 text-green-600" />}
            </button>
          ))}
        </div>
      </SheetContent>
    </Sheet>
  );

  // Mobile Action Sheet for Visibility
  const MobileVisibilitySheet = () => (
    <Sheet open={showVisibilitySheet} onOpenChange={setShowVisibilitySheet}>
      <SheetContent side="bottom" className="px-4 pb-6">
        <SheetHeader>
          <SheetTitle>{t("setting.preference-section.default-memo-visibility")}</SheetTitle>
        </SheetHeader>
        <div className="mt-4">
          {VISIBILITY_OPTIONS.map((option) => (
            <button
              key={option.value}
              onClick={() => handleDefaultMemoVisibilityChanged(convertVisibilityToString(option.value))}
              className={cn(
                "w-full flex items-center justify-between px-4 py-3 border-b border-border last:border-0",
                "active:bg-muted/50",
                setting.memoVisibility === convertVisibilityToString(option.value) && "bg-muted/30",
              )}
            >
              <div className="flex items-center gap-2">
                <VisibilityIcon visibility={option.value} />
                {/* biome-ignore lint/suspicious/noExplicitAny: i18n dynamic key access */}
                <span className="text-sm">{(t as any)(option.labelKey)}</span>
              </div>
              {setting.memoVisibility === convertVisibilityToString(option.value) && <CheckIcon className="w-5 h-5 text-green-600" />}
            </button>
          ))}
        </div>
      </SheetContent>
    </Sheet>
  );

  return (
    <SettingSection>
      <SettingGroup title={t("common.basic")}>
        {isMobile ? (
          <div
            className="w-full flex items-center justify-between px-4 py-3 border-b border-border last:border-0 active:bg-muted/50"
            onClick={() => setShowLocaleSheet(true)}
          >
            <span className="text-sm">{t("common.language")}</span>
            <span className="text-sm text-muted-foreground flex items-center gap-2">
              {setting.locale}
              <ChevronRightIcon className="w-4 h-4" />
            </span>
          </div>
        ) : (
          <SettingRow label={t("common.language")}>
            <LocaleSelect value={setting.locale} onChange={handleLocaleSelectChange} />
          </SettingRow>
        )}

        {isMobile ? (
          <div
            className="w-full flex items-center justify-between px-4 py-3 border-b border-border last:border-0 active:bg-muted/50"
            onClick={() => setShowThemeSheet(true)}
          >
            <span className="text-sm">{t("setting.preference-section.theme")}</span>
            <span className="text-sm text-muted-foreground flex items-center gap-2">
              {THEME_OPTIONS.find((opt) => opt.value === setting.theme)?.label || setting.theme}
              <ChevronRightIcon className="w-4 h-4" />
            </span>
          </div>
        ) : (
          <SettingRow label={t("setting.preference-section.theme")}>
            <ThemeSelect value={setting.theme} onValueChange={handleThemeChange} />
          </SettingRow>
        )}
      </SettingGroup>

      <SettingGroup title={t("setting.preference")} showSeparator>
        {isMobile ? (
          <div
            className="w-full flex items-center justify-between px-4 py-3 border-b border-border last:border-0 active:bg-muted/50"
            onClick={() => setShowVisibilitySheet(true)}
          >
            <span className="text-sm">{t("setting.preference-section.default-memo-visibility")}</span>
            <span className="text-sm text-muted-foreground flex items-center gap-2">
              <VisibilityIcon visibility={convertVisibilityFromString(setting.memoVisibility)} />
              <ChevronRightIcon className="w-4 h-4" />
            </span>
          </div>
        ) : (
          <SettingRow label={t("setting.preference-section.default-memo-visibility")}>
            <Select value={setting.memoVisibility || "PRIVATE"} onValueChange={handleDefaultMemoVisibilityChanged}>
              <SelectTrigger className="min-w-fit">
                <div className="flex items-center gap-2">
                  <VisibilityIcon visibility={convertVisibilityFromString(setting.memoVisibility)} />
                  <SelectValue />
                </div>
              </SelectTrigger>
              <SelectContent>
                {[Visibility.PRIVATE, Visibility.PROTECTED, Visibility.PUBLIC]
                  .map((v) => convertVisibilityToString(v))
                  .map((item) => (
                    <SelectItem key={item} value={item} className="whitespace-nowrap">
                      {t(`memo.visibility.${item.toLowerCase() as Lowercase<typeof item>}`)}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </SettingRow>
        )}
      </SettingGroup>

      {/* Hide Webhook section on mobile - dev tool */}
      {!isMobile && (
        <SettingGroup showSeparator>
          <WebhookSection />
        </SettingGroup>
      )}

      {/* Mobile Sheets */}
      {isMobile && (
        <>
          <MobileLocaleSheet />
          <MobileThemeSheet />
          <MobileVisibilitySheet />
        </>
      )}
    </SettingSection>
  );
};

export default PreferencesSection;
