import {
  BarChart3Icon,
  ChevronRightIcon,
  CogIcon,
  DatabaseIcon,
  KeyIcon,
  LibraryIcon,
  MessageSquareIcon,
  Settings2Icon,
  UserIcon,
  UsersIcon,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import ChatAppsSection from "@/components/Settings/ChatAppsSection";
import InstanceSection from "@/components/Settings/InstanceSection";
import MemberSection from "@/components/Settings/MemberSection";
import MemoRelatedSettings from "@/components/Settings/MemoRelatedSettings";
import { MetricsDashboard } from "@/components/Settings/MetricsDashboard";
import MyAccountSection from "@/components/Settings/MyAccountSection";
import PreferencesSection from "@/components/Settings/PreferencesSection";
import SSOSection from "@/components/Settings/SSOSection";
import StorageSection from "@/components/Settings/StorageSection";
import { useInstance } from "@/contexts/InstanceContext";
import useCurrentUser from "@/hooks/useCurrentUser";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { InstanceSetting_Key } from "@/types/proto/api/v1/instance_service_pb";
import { User_Role } from "@/types/proto/api/v1/user_service_pb";
import { useTranslate } from "@/utils/i18n";

type SettingSection = "my-account" | "preference" | "member" | "system" | "memo-related" | "storage" | "sso" | "metrics" | "chat-apps";

interface State {
  selectedSection: SettingSection;
}

const PERSONAL_SECTIONS: SettingSection[] = ["my-account", "preference", "chat-apps"];
const ADMIN_SECTIONS: SettingSection[] = ["member", "system", "memo-related", "storage", "sso", "metrics"];

const SECTION_ICON_MAP: Record<SettingSection, React.ElementType> = {
  "my-account": UserIcon,
  preference: CogIcon,
  "chat-apps": MessageSquareIcon,
  member: UsersIcon,
  system: Settings2Icon,
  "memo-related": LibraryIcon,
  storage: DatabaseIcon,
  sso: KeyIcon,
  metrics: BarChart3Icon,
};

const SECTION_TITLE_MAP: Record<SettingSection, string> = {
  "my-account": "setting.my-account",
  preference: "setting.preference",
  "chat-apps": "setting.chat-apps.title",
  member: "setting.member-list",
  system: "setting.system-section.title",
  "memo-related": "setting.memo-related-title",
  storage: "setting.storage-section.title",
  sso: "setting.sso-section.title",
  metrics: "setting.metrics-title",
};

const Setting = () => {
  const t = useTranslate();
  const sm = useMediaQuery("sm");
  const location = useLocation();
  const user = useCurrentUser();
  const { profile, fetchSetting } = useInstance();
  const [state, setState] = useState<State>({
    selectedSection: "my-account",
  });
  const isHost = user?.role === User_Role.HOST;

  useEffect(() => {
    let hash = location.hash.slice(1) as SettingSection;
    if (![...PERSONAL_SECTIONS, ...ADMIN_SECTIONS].includes(hash)) {
      hash = "my-account";
    }
    setState({
      selectedSection: hash,
    });
  }, [location.hash]);

  useEffect(() => {
    if (!isHost) {
      return;
    }
    (async () => {
      [InstanceSetting_Key.MEMO_RELATED, InstanceSetting_Key.STORAGE].forEach(async (key) => {
        await fetchSetting(key);
      });
    })();
  }, [isHost, fetchSetting]);

  const handleSectionClick = useCallback((settingSection: SettingSection) => {
    window.location.hash = settingSection;
    setState({ selectedSection: settingSection });
  }, []);

  const renderSection = () => {
    switch (state.selectedSection) {
      case "my-account":
        return <MyAccountSection />;
      case "preference":
        return <PreferencesSection />;
      case "chat-apps":
        return <ChatAppsSection />;
      case "member":
        return <MemberSection />;
      case "system":
        return <InstanceSection />;
      case "memo-related":
        return <MemoRelatedSettings />;
      case "storage":
        return <StorageSection />;
      case "sso":
        return <SSOSection />;
      case "metrics":
        return <MetricsDashboard />;
      default:
        return null;
    }
  };

  const renderMobileSectionCard = (titleKey: string, sections: SettingSection[]) => (
    <div className="mb-6">
      {/* biome-ignore lint/suspicious/noExplicitAny: i18n dynamic key access */}
      <h3 className="text-sm font-semibold text-muted-foreground mb-3 px-4">{(t as any)(titleKey)}</h3>
      <div className="bg-background border border-border rounded-xl overflow-hidden">
        {sections.map((section) => {
          const Icon = SECTION_ICON_MAP[section];
          const titleKey = SECTION_TITLE_MAP[section];
          return (
            <button
              key={section}
              onClick={() => handleSectionClick(section)}
              className={cn(
                "w-full flex items-center gap-3 px-4 py-3 border-b border-border last:border-0",
                "active:bg-muted/50",
                state.selectedSection === section && "bg-muted/30",
              )}
            >
              <Icon className="w-5 h-5 text-muted-foreground shrink-0" />
              <span className="text-sm text-foreground flex-1 text-left">
                {/* biome-ignore lint/suspicious/noExplicitAny: i18n dynamic key access */}
                {section === "chat-apps" ? (t as any)(`setting.${section}.title`) : (t as any)(titleKey)}
              </span>
              <ChevronRightIcon className="w-5 h-5 text-muted-foreground shrink-0" />
            </button>
          );
        })}
      </div>
    </div>
  );

  return (
    <section className="@container w-full max-w-[100rem] min-h-full flex flex-col justify-start items-start pb-8">
      <div className="w-full px-4 sm:px-6">
        {/* Desktop Layout: Sidebar + Content */}
        {sm && (
          <div className="w-full border border-border flex flex-row justify-start items-start px-4 py-3 rounded-xl bg-background text-muted-foreground">
            <div className="flex flex-col justify-start items-start w-40 h-auto shrink-0 py-2">
              <span className="text-sm mt-0.5 pl-3 font-mono select-none text-muted-foreground">{t("common.basic")}</span>
              <div className="w-full flex flex-col justify-start items-start mt-1">
                {PERSONAL_SECTIONS.map((item) => (
                  <button
                    key={item}
                    onClick={() => handleSectionClick(item)}
                    className={cn(
                      "w-full text-left px-3 py-1.5 rounded-md text-sm hover:bg-muted/50 transition-colors",
                      state.selectedSection === item && "bg-muted/30",
                    )}
                  >
                    {item === "chat-apps" ? t(`setting.${item}.title`) : t(`setting.${item}`)}
                  </button>
                ))}
              </div>
              {isHost ? (
                <>
                  <span className="text-sm mt-4 pl-3 font-mono select-none text-muted-foreground">{t("common.admin")}</span>
                  <div className="w-full flex flex-col justify-start items-start mt-1">
                    {ADMIN_SECTIONS.map((item) => (
                      <button
                        key={item}
                        onClick={() => handleSectionClick(item)}
                        className={cn(
                          "w-full text-left px-3 py-1.5 rounded-md text-sm hover:bg-muted/50 transition-colors",
                          state.selectedSection === item && "bg-muted/30",
                        )}
                      >
                        {t(`setting.${item}`)}
                      </button>
                    ))}
                    <span className="px-3 mt-2 opacity-70 text-sm">
                      {t("setting.version")}: v{profile.version}
                    </span>
                  </div>
                </>
              ) : null}
            </div>
            <div className="w-full grow sm:pl-4 overflow-x-auto">{renderSection()}</div>
          </div>
        )}

        {/* Mobile Layout: Grouped card navigation + full-width content */}
        {!sm && (
          <>
            {renderMobileSectionCard("setting.personal", PERSONAL_SECTIONS)}
            {isHost && renderMobileSectionCard("setting.admin", ADMIN_SECTIONS)}
            <div className="mt-4">{renderSection()}</div>
          </>
        )}
      </div>
    </section>
  );
};

export default Setting;
