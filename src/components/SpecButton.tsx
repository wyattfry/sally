type SpecButtonProps = {
  onClick: () => void;
};

export function SpecButton({ onClick }: SpecButtonProps) {
  return (
    <div className="spec-control">
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
