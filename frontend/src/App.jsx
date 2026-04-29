import { GreetService } from "../bindings/changeme";

function App() {
  const hidePopup = () => {
    GreetService.HidePopup().catch((err) => {
      console.error(err);
    });
  };

  return (
    <div className="popup-shell">
      <div className="popup-header">
        <div className="popup-title">Sticker Picker</div>
        <button className="popup-close" onClick={hidePopup} aria-label="Close popup">
          X
        </button>
      </div>
      <div className="popup-content">
        <p>Popup is ready. Press Cmd+Option+Shift+M to open from anywhere.</p>
      </div>
    </div>
  );
}

export default App;
