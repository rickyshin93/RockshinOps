import { useCallback, useId, useRef, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

export type CleanupConfirmationRequest = { message: string; confirm: () => void; cancel?: () => void; permanent?: boolean; details?: ReactNode };

export function CleanupConfirmation({ request, onClose }: { request: CleanupConfirmationRequest; onClose: () => void }) {
  const { t } = useTranslation();
  const title = useId();
  const detail = useId();
  const released = useRef(false);
  const cancel = request.cancel;
  const discard = useCallback(() => {
    if (released.current) return;
    released.current = true;
    cancel?.();
  }, [cancel]);
  const showConfirmation = useCallback((element: HTMLDialogElement | null) => {
    if (!element) return;
    if (!element.open) element.showModal();
    return () => queueMicrotask(() => {
      if (!element.isConnected && element.returnValue !== "confirm") discard();
    });
  }, [discard]);

  return <dialog className="cleanup-dialog" aria-labelledby={title} aria-describedby={detail}
    ref={showConfirmation} onClose={event => {
      if (event.currentTarget.returnValue !== "confirm") discard();
      onClose();
    }}>
    <h2 id={title}>{request.permanent ? t("清空废纸篓") : t("确认移入废纸篓")}</h2>
    <p id={detail}>{request.message}</p>
    {request.details}
    <div>
      <button className="secondary-button" autoFocus onClick={event => event.currentTarget.closest("dialog")?.close()}>{t("取消")}</button>
      <button className="primary-button" data-permanent={request.permanent || undefined} onClick={event => {
        event.currentTarget.closest("dialog")?.close("confirm");
        request.confirm();
      }}>{request.permanent ? t("永久删除") : t("确认移动")}</button>
    </div>
  </dialog>;
}
