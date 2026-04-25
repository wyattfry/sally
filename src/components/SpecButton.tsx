type SpecButtonProps = {
  onClick: () => void;
  itemCount: number;
  onViewItems: () => void;
};

export function SpecButton({ onClick, itemCount, onViewItems }: SpecButtonProps) {
  return (
    <div className="spec-control">
      {itemCount > 0 ? (
        <button
          className="view-items-button"
          type="button"
          title="View schedule items"
          onClick={onViewItems}
        >
          {itemCount}
        </button>
      ) : null}
      <button
        className="spec-button"
        type="button"
        onClick={onClick}
      >
        SPEC
      </button>
    </div>
  );
}
