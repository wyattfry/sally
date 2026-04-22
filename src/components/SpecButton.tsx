type SpecButtonProps = {
  isCurrentPageSpecd?: boolean;
  itemCount: number;
  onClick: () => void;
};

export function SpecButton({ isCurrentPageSpecd = false, itemCount, onClick }: SpecButtonProps) {
  return (
    <div className={isCurrentPageSpecd ? "spec-control spec-control--specd" : "spec-control"}>
      <div
        className={isCurrentPageSpecd ? "spec-context spec-context--specd" : "spec-context"}
        aria-live="polite"
      >
        <div className="spec-project">Sally PoC</div>
        <div className="spec-count">
          {itemCount} {itemCount === 1 ? "item" : "items"}
        </div>
        {isCurrentPageSpecd ? <div className="spec-page-status">Page spec'd</div> : null}
      </div>
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
