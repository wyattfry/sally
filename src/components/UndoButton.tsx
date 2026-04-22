type UndoButtonProps = {
  onClick: () => void;
};

export function UndoButton({ onClick }: UndoButtonProps) {
  return (
    <button className="action-button secondary" type="button" onClick={onClick}>
      Undo
    </button>
  );
}

