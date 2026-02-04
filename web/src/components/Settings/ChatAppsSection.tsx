import { CheckIcon, Loader2Icon, PlusIcon, Trash2Icon, WebhookIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import useIsMobile from "@/hooks/useIsMobile";
import { Platform } from "@/types/proto/api/v1/chat_app_service_pb";
import { useTranslate } from "@/utils/i18n";
import MobileTableCard, { MobileTableCardColumn } from "./MobileTableCard";

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

const PLATFORM_LABELS: Record<number, string> = {
  [Platform.UNSPECIFIED]: "Unknown",
  [Platform.TELEGRAM]: "Telegram",
  [Platform.WHATSAPP]: "WhatsApp",
  [Platform.DINGTALK]: "DingTalk",
};

const ChatAppsSection = ({ className }: ChatAppsSectionProps) => {
  const t = useTranslate();
  const isMobile = useIsMobile();
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const [webhookInfo, setWebhookInfo] = useState<{ webhook_url: string; setup_instructions: string } | null>(null);

  // Form state
  const [newPlatform, setNewPlatform] = useState<Platform>(Platform.TELEGRAM);
  const [newPlatformUserId, setNewPlatformUserId] = useState("");
  const [newAccessToken, setNewAccessToken] = useState("");
  const [newWebhookUrl, setNewWebhookUrl] = useState("");

  // Mobile card columns
  const mobileColumns: MobileTableCardColumn<Credential>[] = [
    {
      key: "platform",
      label: t("setting.chat-apps.platform"),
      render: (_, cred) => (
        <div className="flex items-center gap-2">
          <span className="font-medium">{PLATFORM_LABELS[cred.platform] || cred.platform}</span>
          {cred.enabled ? (
            <span className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
              <CheckIcon className="w-3 h-3" />
              {t("setting.chat-apps.enabled")}
            </span>
          ) : (
            <span className="text-xs text-muted-foreground">{t("setting.chat-apps.disabled")}</span>
          )}
        </div>
      ),
    },
    {
      key: "platformUserId",
      label: t("setting.chat-apps.platform-user-id"),
      render: (_, cred) => <span className="font-mono text-xs">{cred.platformUserId}</span>,
    },
    {
      key: "createdTs",
      label: t("setting.chat-apps.created-at"),
      render: (_, cred) => new Date(cred.createdTs * 1000).toLocaleString(),
    },
  ];

  // Desktop columns (keep existing)
  const desktopRenderItem = (cred: Credential) => (
    <div key={cred.id} className="border border-border rounded-lg p-4 bg-background">
      <div className="flex flex-row justify-between items-start">
        <div className="flex-1">
          <div className="flex flex-row items-center gap-2 mb-2">
            <h3 className="font-medium">{PLATFORM_LABELS[cred.platform] || cred.platform}</h3>
            <span className="text-xs text-muted-foreground px-2 py-0.5 bg-muted rounded">{cred.platformUserId}</span>
            {cred.enabled ? (
              <span className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
                <CheckIcon className="w-3 h-3" />
                {t("setting.chat-apps.enabled")}
              </span>
            ) : (
              <span className="text-xs text-muted-foreground">{t("setting.chat-apps.disabled")}</span>
            )}
          </div>
          <p className="text-sm text-muted-foreground">
            {t("setting.chat-apps.created-at")}: {new Date(cred.createdTs * 1000).toLocaleString()}
          </p>
        </div>
        <div className="flex flex-row gap-2 items-center">
          <Button variant="ghost" size="sm" onClick={() => handleGetWebhookInfo(cred.platform)}>
            <WebhookIcon className="w-4 h-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={() => handleToggleEnabled(cred.platform, !cred.enabled)}>
            {cred.enabled ? t("setting.chat-apps.disable") : t("setting.chat-apps.enable")}
          </Button>
          <Button variant="ghost" size="sm" onClick={() => handleDelete(cred.platform)}>
            <Trash2Icon className="w-4 h-4 text-destructive" />
          </Button>
        </div>
      </div>
    </div>
  );

  // Fetch credentials on mount
  useEffect(() => {
    fetchCredentials();
  }, []);

  // Fetch credentials
  const fetchCredentials = async () => {
    setIsLoading(true);
    setErrorMessage(null);
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
      setErrorMessage(t("setting.chat-apps.fetch-failed"));
    } finally {
      setIsLoading(false);
    }
  };

  // Register credential
  const handleRegister = async () => {
    if (!newPlatformUserId || (!newAccessToken && newPlatform !== Platform.WHATSAPP)) {
      setErrorMessage(t("setting.chat-apps.validation-error"));
      return;
    }

    setIsSubmitting(true);
    setErrorMessage(null);
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

      // Reset form and close dialog
      setNewPlatformUserId("");
      setNewAccessToken("");
      setNewWebhookUrl("");
      setShowAddDialog(false);

      // Refresh credentials
      await fetchCredentials();
    } catch (error) {
      console.error("Failed to register credential:", error);
      setErrorMessage(t("setting.chat-apps.register-failed"));
    } finally {
      setIsSubmitting(false);
    }
  };

  // Delete credential
  const handleDelete = async (platform: Platform) => {
    setIsSubmitting(true);
    setErrorMessage(null);
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

      // Refresh credentials
      await fetchCredentials();
    } catch (error) {
      console.error("Failed to delete credential:", error);
      setErrorMessage(t("setting.chat-apps.delete-failed"));
    } finally {
      setIsSubmitting(false);
    }
  };

  // Toggle enabled state
  const handleToggleEnabled = async (platform: Platform, enabled: boolean) => {
    setErrorMessage(null);
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
      setErrorMessage(t("setting.chat-apps.update-failed"));
    }
  };

  // Get webhook info
  const handleGetWebhookInfo = async (platform: Platform) => {
    try {
      const response = await fetch(`/api/v1/chat-apps/webhook-info/${Platform[platform].toLowerCase()}`, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("access_token")}`,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to get webhook info");
      }

      const data = await response.json();
      setWebhookInfo({
        webhook_url: data.webhook_url,
        setup_instructions: data.setup_instructions,
      });
    } catch (error) {
      console.error("Failed to get webhook info:", error);
    }
  };

  return (
    <div className={className}>
      <div className="flex flex-row justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">{t("setting.chat-apps.title")}</h2>
        <Button variant="outline" size="sm" onClick={() => setShowAddDialog(true)}>
          <PlusIcon className="w-4 h-4 mr-2" />
          {t("setting.chat-apps.add")}
        </Button>
      </div>

      <p className="text-sm text-muted-foreground mb-4">{t("setting.chat-apps.description")}</p>

      {errorMessage && (
        <div className="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-md text-sm text-destructive">{errorMessage}</div>
      )}

      {isLoading ? (
        <div className="flex justify-center py-8">
          <Loader2Icon className="w-6 h-6 animate-spin text-muted-foreground" />
        </div>
      ) : credentials.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          <p>{t("setting.chat-apps.no-credentials")}</p>
        </div>
      ) : isMobile ? (
        /* Mobile: Use card layout */
        <MobileTableCard
          columns={mobileColumns}
          data={credentials}
          emptyMessage={t("setting.chat-apps.no-credentials")}
          getRowKey={(cred) => String(cred.id)}
          renderActions={(cred) => (
            <>
              <Button variant="ghost" size="sm" onClick={() => handleGetWebhookInfo(cred.platform)} className="h-9 px-3">
                <WebhookIcon className="w-4 h-4" />
              </Button>
              <Button variant="ghost" size="sm" onClick={() => handleToggleEnabled(cred.platform, !cred.enabled)} className="h-9 px-3">
                {cred.enabled ? t("setting.chat-apps.disable") : t("setting.chat-apps.enable")}
              </Button>
              <Button variant="ghost" size="sm" onClick={() => handleDelete(cred.platform)} className="h-9 px-3">
                <Trash2Icon className="w-4 h-4 text-destructive" />
              </Button>
            </>
          )}
        />
      ) : (
        /* Desktop: Use existing card layout */
        <div className="space-y-3">{credentials.map((cred) => desktopRenderItem(cred))}</div>
      )}

      {/* Add Credential Dialog - use Sheet on mobile */}
      {isMobile ? (
        <Sheet open={showAddDialog} onOpenChange={setShowAddDialog}>
          <SheetContent side="bottom" className="h-[85vh]">
            <SheetHeader>
              <SheetTitle>{t("setting.chat-apps.add-credential")}</SheetTitle>
              <SheetDescription>{t("setting.chat-apps.add-description")}</SheetDescription>
            </SheetHeader>

            <div className="space-y-4 py-4">
              {/* Platform Selection */}
              <div className="space-y-2">
                <Label htmlFor="platform">{t("setting.chat-apps.platform")}</Label>
                <Select value={String(newPlatform)} onValueChange={(v) => setNewPlatform(Number(v) as Platform)}>
                  <SelectTrigger id="platform">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={String(Platform.TELEGRAM)}>Telegram</SelectItem>
                    <SelectItem value={String(Platform.WHATSAPP)}>WhatsApp</SelectItem>
                    <SelectItem value={String(Platform.DINGTALK)}>DingTalk</SelectItem>
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
                    newPlatform === Platform.TELEGRAM ? "123456789" : newPlatform === Platform.DINGTALK ? "manager1234" : "user_id"
                  }
                />
                <p className="text-xs text-muted-foreground">
                  {newPlatform === Platform.TELEGRAM && t("setting.chat-apps.telegram-user-id-hint")}
                  {newPlatform === Platform.DINGTALK && t("setting.chat-apps.dingtalk-user-id-hint")}
                </p>
              </div>

              {/* Access Token */}
              <div className="space-y-2">
                <Label htmlFor="accessToken">
                  {newPlatform === Platform.WHATSAPP ? "Bridge API Key (Optional)" : t("setting.chat-apps.access-token")}
                </Label>
                <Input
                  id="accessToken"
                  type="password"
                  value={newAccessToken}
                  onChange={(e) => setNewAccessToken(e.target.value)}
                  placeholder={newPlatform === Platform.TELEGRAM ? "123456789:ABCDefGhIJKlMnOPqrstUVwxYZ" : "your_token_here"}
                />
                <p className="text-xs text-muted-foreground">
                  {newPlatform === Platform.TELEGRAM && t("setting.chat-apps.telegram-token-hint")}
                  {newPlatform === Platform.DINGTALK && t("setting.chat-apps.dingtalk-token-hint")}
                  {newPlatform === Platform.WHATSAPP && "Leave empty if bridge does not require API key"}
                </p>
              </div>

              {/* Webhook URL (WhatsApp and DingTalk) */}
              {(newPlatform === Platform.DINGTALK || newPlatform === Platform.WHATSAPP) && (
                <div className="space-y-2">
                  <Label htmlFor="webhookUrl">
                    {newPlatform === Platform.WHATSAPP ? "Bridge URL" : t("setting.chat-apps.webhook-url")}
                  </Label>
                  <Input
                    id="webhookUrl"
                    value={newWebhookUrl}
                    onChange={(e) => setNewWebhookUrl(e.target.value)}
                    placeholder={
                      newPlatform === Platform.WHATSAPP ? "http://localhost:3001" : "https://oapi.dingtalk.com/robot/send?access_token=..."
                    }
                  />
                  <p className="text-xs text-muted-foreground">
                    {newPlatform === Platform.WHATSAPP ? "URL of the Baileys Bridge service" : t("setting.chat-apps.dingtalk-webhook-hint")}
                  </p>
                </div>
              )}
            </div>

            <SheetFooter>
              <Button variant="outline" onClick={() => setShowAddDialog(false)}>
                {t("common.cancel")}
              </Button>
              <Button onClick={handleRegister} disabled={isSubmitting}>
                {isSubmitting && <Loader2Icon className="w-4 h-4 mr-2 animate-spin" />}
                {t("common.confirm")}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      ) : (
        /* Desktop: Use Dialog */
        <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
          <DialogContent className="max-w-[28rem]">
            <DialogHeader>
              <DialogTitle>{t("setting.chat-apps.add-credential")}</DialogTitle>
              <DialogDescription>{t("setting.chat-apps.add-description")}</DialogDescription>
            </DialogHeader>

            <div className="space-y-4 py-4">
              {/* Platform Selection */}
              <div className="space-y-2">
                <Label htmlFor="platform">{t("setting.chat-apps.platform")}</Label>
                <Select value={String(newPlatform)} onValueChange={(v) => setNewPlatform(Number(v) as Platform)}>
                  <SelectTrigger id="platform">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={String(Platform.TELEGRAM)}>Telegram</SelectItem>
                    <SelectItem value={String(Platform.WHATSAPP)}>WhatsApp</SelectItem>
                    <SelectItem value={String(Platform.DINGTALK)}>DingTalk</SelectItem>
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
                    newPlatform === Platform.TELEGRAM ? "123456789" : newPlatform === Platform.DINGTALK ? "manager1234" : "user_id"
                  }
                />
                <p className="text-xs text-muted-foreground">
                  {newPlatform === Platform.TELEGRAM && t("setting.chat-apps.telegram-user-id-hint")}
                  {newPlatform === Platform.DINGTALK && t("setting.chat-apps.dingtalk-user-id-hint")}
                </p>
              </div>

              {/* Access Token */}
              <div className="space-y-2">
                <Label htmlFor="accessToken">
                  {newPlatform === Platform.WHATSAPP ? "Bridge API Key (Optional)" : t("setting.chat-apps.access-token")}
                </Label>
                <Input
                  id="accessToken"
                  type="password"
                  value={newAccessToken}
                  onChange={(e) => setNewAccessToken(e.target.value)}
                  placeholder={newPlatform === Platform.TELEGRAM ? "123456789:ABCDefGhIJKlMnOPqrstUVwxYZ" : "your_token_here"}
                />
                <p className="text-xs text-muted-foreground">
                  {newPlatform === Platform.TELEGRAM && t("setting.chat-apps.telegram-token-hint")}
                  {newPlatform === Platform.DINGTALK && t("setting.chat-apps.dingtalk-token-hint")}
                  {newPlatform === Platform.WHATSAPP && "Leave empty if bridge does not require API key"}
                </p>
              </div>

              {/* Webhook URL (WhatsApp and DingTalk) */}
              {(newPlatform === Platform.DINGTALK || newPlatform === Platform.WHATSAPP) && (
                <div className="space-y-2">
                  <Label htmlFor="webhookUrl">
                    {newPlatform === Platform.WHATSAPP ? "Bridge URL" : t("setting.chat-apps.webhook-url")}
                  </Label>
                  <Input
                    id="webhookUrl"
                    value={newWebhookUrl}
                    onChange={(e) => setNewWebhookUrl(e.target.value)}
                    placeholder={
                      newPlatform === Platform.WHATSAPP ? "http://localhost:3001" : "https://oapi.dingtalk.com/robot/send?access_token=..."
                    }
                  />
                  <p className="text-xs text-muted-foreground">
                    {newPlatform === Platform.WHATSAPP ? "URL of the Baileys Bridge service" : t("setting.chat-apps.dingtalk-webhook-hint")}
                  </p>
                </div>
              )}
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => setShowAddDialog(false)}>
                {t("common.cancel")}
              </Button>
              <Button onClick={handleRegister} disabled={isSubmitting}>
                {isSubmitting && <Loader2Icon className="w-4 h-4 mr-2 animate-spin" />}
                {t("common.confirm")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}

      {/* Webhook Info Dialog - use Sheet on mobile */}
      {isMobile ? (
        <Sheet open={!!webhookInfo} onOpenChange={(open) => !open && setWebhookInfo(null)}>
          <SheetContent side="bottom">
            <SheetHeader>
              <SheetTitle>{t("setting.chat-apps.webhook-info")}</SheetTitle>
            </SheetHeader>
            <div className="space-y-4 py-4">
              {webhookInfo && (
                <>
                  <div className="space-y-2">
                    <Label>Webhook URL</Label>
                    <div className="flex gap-2">
                      <Input readOnly value={webhookInfo.webhook_url} />
                      <Button
                        variant="outline"
                        size="icon"
                        onClick={() => {
                          navigator.clipboard.writeText(webhookInfo.webhook_url);
                        }}
                      >
                        <CheckIcon className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                  {webhookInfo.setup_instructions && (
                    <div className="space-y-2">
                      <Label>Instructions</Label>
                      <pre className="text-xs bg-muted p-2 rounded whitespace-pre-wrap">{webhookInfo.setup_instructions}</pre>
                    </div>
                  )}
                </>
              )}
            </div>
            <SheetFooter>
              <Button onClick={() => setWebhookInfo(null)}>{t("common.close")}</Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      ) : (
        /* Desktop: Use Dialog */
        <Dialog open={!!webhookInfo} onOpenChange={(open) => !open && setWebhookInfo(null)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("setting.chat-apps.webhook-info")}</DialogTitle>
            </DialogHeader>
            <div className="space-y-4 py-4">
              {webhookInfo && (
                <>
                  <div className="space-y-2">
                    <Label>Webhook URL</Label>
                    <div className="flex gap-2">
                      <Input readOnly value={webhookInfo.webhook_url} />
                      <Button
                        variant="outline"
                        size="icon"
                        onClick={() => {
                          navigator.clipboard.writeText(webhookInfo.webhook_url);
                        }}
                      >
                        <CheckIcon className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                  {webhookInfo.setup_instructions && (
                    <div className="space-y-2">
                      <Label>Instructions</Label>
                      <pre className="text-xs bg-muted p-2 rounded whitespace-pre-wrap">{webhookInfo.setup_instructions}</pre>
                    </div>
                  )}
                </>
              )}
            </div>
            <DialogFooter>
              <Button onClick={() => setWebhookInfo(null)}>{t("common.close")}</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
};

export default ChatAppsSection;
