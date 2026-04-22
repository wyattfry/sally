type SpecButtonProps = {
  itemCount: number;
  onClick: () => void;
};

export function SpecButton({ itemCount, onClick }: SpecButtonProps) {
  return (
    <div className="spec-control">
      <div className="spec-context" aria-live="polite">
        <div className="spec-project">Sally PoC</div>
        <div className="spec-count">
          {itemCount} {itemCount === 1 ? "item" : "items"}
        </div>
      </div>
      <button className="spec-button" type="button" onClick={onClick}>
        SPEC
      </button>
    </div>
  );
}

