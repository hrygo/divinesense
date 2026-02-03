import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useToast } from "@/hooks/useToast";
import { Platform } from "@/types/proto/api/v1/chat_app_service_pb";
import { useTranslate } from "@/utils/i18n";
import { Trash2Icon, PlusIcon, WebhookIcon, CheckIcon, Loader2Icon } from "lucide-react";
import { useState } from "react";

// Types based on proto
interface Credential {
  id: number;
  userId: number;
  platform: Platform;
  platformUserId: string;
  platformChatId: string;
  enabled: boolean;
  createdTs: number;
  updatedTs: number;
}

interface ChatAppsSectionProps {
  className?: string;
}

const PLATFORM_LABELS: Record<Platform, string> = {
  [Platform.PLATFORM_UNSPECIFIED]: "Unknown",
  [Platform.PLATFORM_TELEGRAM]: "Telegram",
  [Platform.PLATFORM_WHATSAPP]: "WhatsApp",
  [Platform.PLATFORM_DINGTALK]: "DingTalk",
};

const ChatAppsSection = ({ className }: ChatAppsSectionProps) => {
  const t = useTranslate();
  const { toast } = useToast();
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Form state
  const [newPlatform, setNewPlatform] = useState<Platform>(Platform.PLATFORM_TELEGRAM);
  const [newPlatformUserId, setNewPlatformUserId] = useState("");
  const [newAccessToken, setNewAccessToken] = useState("");
  const [newWebhookUrl, setNewWebhookUrl] = useState("");

  // Fetch credentials
  const fetchCredentials = async () => {
    setIsLoading(true);
    try {
      const response = await fetch("/api/v1/chat-apps/credentials", {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("access_token")}`,
        },
      });
      if (!response.ok) {
        throw new Error("Failed to fetch credentials");
      }
      const data = await response.json();
      setCredentials(data.credentials || []);
    } catch (error) {
      console.error("Failed to fetch credentials:", error);
      toast({
        title: t("setting.chat-apps.fetch-failed"),
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  };

  // Register credential
  const handleRegister = async () => {
    if (!newPlatformUserId || !newAccessToken) {
      toast({
        title: t("setting.chat-apps.validation-error"),
        description: t("setting.chat-apps.required-fields"),
        variant: "destructive",
      });
      return;
    }

    setIsSubmitting(true);
    try {
      const response = await fetch("/api/v1/chat-apps/credentials", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("access_token")}`,
        },
        body: JSON.stringify({
          platform: newPlatform,
          platform_user_id: newPlatformUserId,
          platform_chat_id: newPlatformUserId,
          access_token: newAccessToken,
          webhook_url: newWebhookUrl,
        }),
      });

      if (!response.ok) {
        throw new Error("Failed to register credential");
      }

      toast({
        title: t("setting.chat-apps.register-success"),
      });

      // Reset form and close dialog
      setNewPlatformUserId("");
      setNewAccessToken("");
      setNewWebhookUrl("");
      setShowAddDialog(false);

      // Refresh credentials
      await fetchCredentials();
    } catch (error) {
      console.error("Failed to register credential:", error);
      toast({
        title: t("setting.chat-apps.register-failed"),
        variant: "destructive",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  // Delete credential
  const handleDelete = async (platform: Platform) => {
    setIsSubmitting(true);
    try {
      const response = await fetch(`/api/v1/chat-apps/credentials/${platform}`, {
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${localStorage.getItem("access_token")}`,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to delete credential");
      }

      toast({
        title: t("setting.chat-apps.delete-success"),
      });

      // Refresh credentials
      await fetchCredentials();
    } catch (error) {
      console.error("Failed to delete credential:", error);
      toast({
        title: t("setting.chat-apps.delete-failed"),
        variant: "destructive",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  // Toggle enabled state
  const handleToggleEnabled = async (platform: Platform, enabled: boolean) => {
    try {
      const response = await fetch(`/api/v1/chat-apps/credentials/${platform}`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("access_token")}`,
        },
        body: JSON.stringify({
          platform: platform,
          enabled: enabled,
        }),
      });

      if (!response.ok) {
        throw new Error("Failed to update credential");
      }

      // Refresh credentials
      await fetchCredentials();
    } catch (error) {
      console.error("Failed to toggle credential:", error);
      toast({
        title: t("setting.chat-apps.update-failed"),
        variant: "destructive",
      });
    }
  };

  // Get webhook info
  const handleGetWebhookInfo = async (platform: Platform) => {
    try {
      const response = await fetch(
        `/api/v1/chat-apps/webhook-info/${Platform[platform].toLowerCase()}`,
        {
          headers: {
            Authorization: `Bearer ${localStorage.getItem("access_token")}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error("Failed to get webhook info");
      }

      const data = await response.json();

      // Show webhook info in a modal or alert
      toast({
        title: t("setting.chat-apps.webhook-url"),
        description: data.webhook_url,
      });
    } catch (error) {
      console.error("Failed to get webhook info:", error);
    }
  };

  // Initial fetch
  useState(() => {
    fetchCredentials();
  });

  return (
    <div className={className}>
      <div className="flex flex-row justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">{t("setting.chat-apps.title")}</h2>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShowAddDialog(true)}
        >
          <PlusIcon className="w-4 h-4 mr-2" />
          {t("setting.chat-apps.add")}
        </Button>
      </div>

      <p className="text-sm text-muted-foreground mb-4">
        {t("setting.chat-apps.description")}
      </p>

      {isLoading ? (
        <div className="flex justify-center py-8">
          <Loader2Icon className="w-6 h-6 animate-spin text-muted-foreground" />
        </div>
      ) : credentials.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          <p>{t("setting.chat-apps.no-credentials")}</p>
        </div>
      ) : (
        <div className="space-y-3">
          {credentials.map((cred) => (
            <div
              key={cred.id}
              className="border border-border rounded-lg p-4 bg-background"
            >
              <div className="flex flex-row justify-between items-start">
                <div className="flex-1">
                  <div className="flex flex-row items-center gap-2 mb-2">
                    <h3 className="font-medium">
                      {PLATFORM_LABELS[cred.platform] || cred.platform}
                    </h3>
                    <span className="text-xs text-muted-foreground px-2 py-0.5 bg-muted rounded">
                      {cred.platformUserId}
                    </span>
                    {cred.enabled ? (
                      <span className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
                        <CheckIcon className="w-3 h-3" />
                        {t("setting.chat-apps.enabled")}
                      </span>
                    ) : (
                      <span className="text-xs text-muted-foreground">
                        {t("setting.chat-apps.disabled")}
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {t("setting.chat-apps.created-at")}:{" "}
                    {new Date(cred.createdTs * 1000).toLocaleString()}
                  </p>
                </div>
                <div className="flex flex-row gap-2 items-center">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleGetWebhookInfo(cred.platform)}
                  >
                    <WebhookIcon className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleToggleEnabled(cred.platform, !cred.enabled)}
                  >
                    {cred.enabled
                      ? t("setting.chat-apps.disable")
                      : t("setting.chat-apps.enable")}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDelete(cred.platform)}
                  >
                    <Trash2Icon className="w-4 h-4 text-destructive" />
                  </Button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Add Credential Dialog */}
      <AlertDialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <AlertDialogContent className="max-w-[28rem]">
          <AlertDialogHeader>
            <AlertDialogTitle>{t("setting.chat-apps.add-credential")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("setting.chat-apps.add-description")}
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="space-y-4 py-4">
            {/* Platform Selection */}
            <div className="space-y-2">
              <Label htmlFor="platform">{t("setting.chat-apps.platform")}</Label>
              <Select
                value={String(newPlatform)}
                onValueChange={(v) => setNewPlatform(Number(v) as Platform)}
              >
                <SelectTrigger id="platform">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={String(Platform.PLATFORM_TELEGRAM)}>
                    Telegram
                  </SelectItem>
                  <SelectItem value={String(Platform.PLATFORM_WHATSAPP)}>
                    WhatsApp
                  </SelectItem>
                  <SelectItem value={String(Platform.PLATFORM_DINGTALK)}>
                    DingTalk
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Platform User ID */}
            <div className="space-y-2">
              <Label htmlFor="platformUserId">{t("setting.chat-apps.platform-user-id")}</Label>
              <Input
                id="platformUserId"
                value={newPlatformUserId}
                onChange={(e) => setNewPlatformUserId(e.target.value)}
                placeholder={
                  newPlatform === Platform.PLATFORM_TELEGRAM
                    ? "123456789"
                    : newPlatform === Platform.PLATFORM_DINGTALK
                      ? "manager1234"
                      : "user_id"
                }
              />
              <p className="text-xs text-muted-foreground">
                {newPlatform === Platform.PLATFORM_TELEGRAM &&
                  t("setting.chat-apps.telegram-user-id-hint")}
                {newPlatform === Platform.PLATFORM_DINGTALK &&
                  t("setting.chat-apps.dingtalk-user-id-hint")}
              </p>
            </div>

            {/* Access Token */}
            <div className="space-y-2">
              <Label htmlFor="accessToken">{t("setting.chat-apps.access-token")}</Label>
              <Input
                id="accessToken"
                type="password"
                value={newAccessToken}
                onChange={(e) => setNewAccessToken(e.target.value)}
                placeholder={
                  newPlatform === Platform.PLATFORM_TELEGRAM
                    ? "123456789:ABCDefGhIJKlMnOPqrstUVwxYZ"
                    : "your_token_here"
                }
              />
              <p className="text-xs text-muted-foreground">
                {newPlatform === Platform.PLATFORM_TELEGRAM &&
                  t("setting.chat-apps.telegram-token-hint")}
                {newPlatform === Platform.PLATFORM_DINGTALK &&
                  t("setting.chat-apps.dingtalk-token-hint")}
              </p>
            </div>

            {/* Webhook URL (DingTalk only) */}
            {newPlatform === Platform.PLATFORM_DINGTALK && (
              <div className="space-y-2">
                <Label htmlFor="webhookUrl">{t("setting.chat-apps.webhook-url")}</Label>
                <Input
                  id="webhookUrl"
                  value={newWebhookUrl}
                  onChange={(e) => setNewWebhookUrl(e.target.value)}
                  placeholder="https://oapi.dingtalk.com/robot/send?access_token=..."
                />
                <p className="text-xs text-muted-foreground">
                  {t("setting.chat-apps.dingtalk-webhook-hint")}
                </p>
              </div>
            )}
          </div>

          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="outline">{t("common.cancel")}</Button>
            </AlertDialogCancel>
            <Button onClick={handleRegister} disabled={isSubmitting}>
              {isSubmitting && <Loader2Icon className="w-4 h-4 mr-2 animate-spin" />}
              {t("common.confirm")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default ChatAppsSection;
