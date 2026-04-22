type SpecButtonProps = {
  isCurrentPageSpecd?: boolean;
  itemCount: number;
  onOpenSchedule: () => void;
  onClick: () => void;
};

export function SpecButton({
  isCurrentPageSpecd = false,
  itemCount,
  onOpenSchedule,
  onClick
}: SpecButtonProps) {
  return (
    <div className={isCurrentPageSpecd ? "spec-control spec-control--specd" : "spec-control"}>
      <button
        className={isCurrentPageSpecd ? "spec-context spec-context--specd" : "spec-context"}
        aria-live="polite"
        type="button"
        onClick={onOpenSchedule}
      >
        <div className="spec-project">Sally PoC</div>
        <div className="spec-count">
          {itemCount} {itemCount === 1 ? "item" : "items"}
        </div>
        {isCurrentPageSpecd ? <div className="spec-page-status">Page spec'd</div> : null}
      </button>
      <button
        className={isCurrentPageSpecd ? "spec-button spec-button--specd" : "spec-button"}
        type="button"
        onClick={onClick}
      >
        SPEC
      </button>
    </div>
  );
}
