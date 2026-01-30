import { Check, ChevronLeft, ChevronRight, X } from "lucide-react";
import { memo, useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

/**
 * Phase 3: 多步向导工具
 *
 * 用于收集复杂的结构化数据，支持：
 * - 分步输入验证
 * - 进度保存与恢复
 * - 上一步/下一步导航
 * - 完成预览
 */

export interface WizardStep {
  id: string;
  title: string;
  description?: string;
  component: React.ComponentType<WizardStepProps>;
  validate?: (data: Record<string, unknown>) => boolean;
  skipIf?: (data: Record<string, unknown>) => boolean;
}

export interface WizardStepProps {
  data: Record<string, unknown>;
  onChange: (key: string, value: unknown) => void;
  onNext: () => void;
  onBack: () => void;
  isLastStep: boolean;
}

export interface MultiStepWizardProps {
  steps: WizardStep[];
  initialData?: Record<string, unknown>;
  onComplete: (data: Record<string, unknown>) => void;
  onCancel: () => void;
  onDismiss?: () => void;
  className?: string;
}

const MultiStepWizard = memo(function MultiStepWizard({ steps, initialData = {}, onComplete, onCancel, className }: MultiStepWizardProps) {
  const { t } = useTranslation();
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [stepData, setStepData] = useState<Record<string, unknown>>(initialData);
  const [completedSteps, setCompletedSteps] = useState<Set<number>>(new Set());
  const [validationErrors, setValidationErrors] = useState<Record<number, string>>({});

  const currentStep = steps[currentStepIndex];
  const isLastStep = currentStepIndex === steps.length - 1;
  const isFirstStep = currentStepIndex === 0;

  // 检查当前步骤是否应该跳过
  const shouldSkipCurrentStep = currentStep?.skipIf?.(stepData);

  // 自动跳过应该跳过的步骤
  if (shouldSkipCurrentStep && !isLastStep) {
    setCurrentStepIndex((prev) => prev + 1);
  }

  // 处理数据变更
  const handleDataChange = useCallback(
    (key: string, value: unknown) => {
      setStepData((prev) => ({
        ...prev,
        [key]: value,
      }));
      // 清除该步骤的验证错误
      if (validationErrors[currentStepIndex]) {
        setValidationErrors((prev) => {
          const copy = { ...prev };
          delete copy[currentStepIndex];
          return copy;
        });
      }
    },
    [currentStepIndex, validationErrors],
  );

  // 验证当前步骤
  const validateCurrentStep = useCallback((): boolean => {
    if (!currentStep.validate) return true;

    const isValid = currentStep.validate!(stepData);
    if (!isValid) {
      setValidationErrors((prev) => ({
        ...prev,
        [currentStepIndex]: t("wizard.validation-error") || "请填写必填项",
      }));
    }
    return isValid;
  }, [currentStep, stepData, t]);

  // 下一步
  const handleNext = useCallback(() => {
    if (!validateCurrentStep()) return;

    // 标记当前步骤为已完成
    setCompletedSteps((prev) => new Set(prev).add(currentStepIndex));

    if (isLastStep) {
      onComplete(stepData);
    } else {
      setCurrentStepIndex((prev) => prev + 1);
    }
  }, [validateCurrentStep, currentStepIndex, isLastStep, stepData, onComplete]);

  // 上一步
  const handleBack = useCallback(() => {
    setCurrentStepIndex((prev) => Math.max(0, prev - 1));
  }, []);

  // 跳转到指定步骤
  const handleStepClick = useCallback(
    (index: number) => {
      // 只能跳转到已完成的步骤或下一步
      if (index <= currentStepIndex || completedSteps.has(index)) {
        setCurrentStepIndex(index);
      }
    },
    [currentStepIndex, completedSteps],
  );

  // 进度百分比
  const progressPercent = ((currentStepIndex + 1) / steps.length) * 100;

  const CurrentStepComponent = currentStep.component;

  return (
    <div
      className={cn(
        "rounded-2xl border bg-card shadow-lg overflow-hidden",
        "animate-in fade-in slide-in-from-bottom-4 duration-300",
        className,
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-foreground">{currentStep.title}</span>
          <span className="text-xs text-muted-foreground">
            {currentStepIndex + 1} / {steps.length}
          </span>
        </div>
        <button onClick={onCancel} className="p-1 rounded-md hover:bg-muted transition-colors" aria-label="Cancel">
          <X className="w-4 h-4 text-muted-foreground" />
        </button>
      </div>

      {/* Progress Bar */}
      <div className="h-1 bg-muted">
        <div className="h-full bg-primary transition-all duration-300 ease-out" style={{ width: `${progressPercent}%` }} />
      </div>

      {/* Step Indicators */}
      <div className="flex items-center justify-center gap-1 px-4 py-2 bg-muted/20">
        {steps.map((step, index) => {
          const isCompleted = completedSteps.has(index);
          const isCurrent = index === currentStepIndex;
          const canNavigate = index <= currentStepIndex || isCompleted;

          return (
            <button
              key={step.id}
              onClick={() => canNavigate && handleStepClick(index)}
              disabled={!canNavigate}
              className={cn(
                "w-6 h-6 rounded-full flex items-center justify-center text-[10px] font-medium transition-all",
                isCurrent && "bg-primary text-primary-foreground scale-110",
                isCompleted && "bg-green-500 text-white",
                !isCurrent && !isCompleted && "bg-muted text-muted-foreground",
                canNavigate && !isCurrent && "hover:bg-muted-foreground hover:text-background cursor-pointer",
                !canNavigate && "opacity-50 cursor-not-allowed",
              )}
              aria-label={`Step ${index + 1}: ${step.title}`}
            >
              {isCompleted ? <Check className="w-3 h-3" /> : index + 1}
            </button>
          );
        })}
      </div>

      {/* Current Step Content */}
      <div className="p-4">
        {currentStep.description && <p className="text-sm text-muted-foreground mb-4">{currentStep.description}</p>}

        <CurrentStepComponent data={stepData} onChange={handleDataChange} onNext={handleNext} onBack={handleBack} isLastStep={isLastStep} />

        {validationErrors[currentStepIndex] && <p className="mt-2 text-sm text-destructive">{validationErrors[currentStepIndex]}</p>}
      </div>

      {/* Footer Navigation */}
      <div className="flex items-center justify-between px-4 py-3 border-t bg-muted/30">
        <button
          onClick={handleBack}
          disabled={isFirstStep}
          className={cn(
            "flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors",
            isFirstStep ? "text-muted-foreground cursor-not-allowed" : "text-foreground hover:bg-muted",
          )}
        >
          <ChevronLeft className="w-4 h-4" />
          {t("wizard.back") || "上一步"}
        </button>

        <button
          onClick={handleNext}
          className={cn(
            "flex items-center gap-1 px-4 py-1.5 rounded-lg text-sm font-medium transition-colors",
            "bg-primary text-primary-foreground hover:bg-primary/90",
          )}
        >
          {isLastStep ? t("wizard.complete") || "完成" : t("wizard.next") || "下一步"}
          {!isLastStep && <ChevronRight className="w-4 h-4" />}
        </button>
      </div>
    </div>
  );
});

export default MultiStepWizard;
