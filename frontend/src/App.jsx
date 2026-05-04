import { useEffect, useMemo, useState } from "react";
import { ChevronLeft, ChevronRight, Clock3, Plus, X } from "lucide-react";
import { GreetService } from "../bindings/changeme";

const RECENT_KEY = "wtw:recent-stickers";
const RECENT_MAX = 16;

function App() {
  const [packs, setPacks] = useState([]);
  const [activeNavId, setActiveNavId] = useState("");
  const [stickers, setStickers] = useState([]);
  const [recentStickers, setRecentStickers] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const raw = localStorage.getItem(RECENT_KEY);
    if (!raw) return;
    try {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        setRecentStickers(parsed.slice(0, RECENT_MAX));
      }
    } catch (err) {
      console.error(err);
    }
  }, []);

  useEffect(() => {
    const syncPopupHeight = () => {
      document.documentElement.style.setProperty("--popup-height-real", `${window.innerHeight}px`);
    };
    syncPopupHeight();
    window.addEventListener("resize", syncPopupHeight);
    return () => window.removeEventListener("resize", syncPopupHeight);
  }, []);

  useEffect(() => {
    GreetService.ListStickerPacks()
      .then((packList) => {
        setPacks(packList);
        if (packList.length > 0) {
          setActiveNavId(packList[0].id);
        }
      })
      .catch((err) => console.error(err))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!activeNavId) {
      setStickers([]);
      return;
    }
    if (activeNavId === "recent") {
      setStickers(recentStickers);
      return;
    }
    GreetService.GetPackStickers(activeNavId)
      .then((items) => {
        setStickers(items);
      })
      .catch((err) => {
        console.error(err);
        setStickers([]);
      });
  }, [activeNavId, recentStickers]);

  const navItems = useMemo(() => {
    const items = [];
    if (recentStickers.length > 0) {
      items.push({
        id: "recent",
        kind: "recent",
        title: "Recent",
      });
    }
    for (const pack of packs) {
      items.push({
        id: pack.id,
        kind: "pack",
        thumbDataUrl: pack.thumbDataUrl,
        title: pack.name,
      });
    }
    return items;
  }, [packs, recentStickers]);

  const activeNavIndex = useMemo(
    () => navItems.findIndex((item) => item.id === activeNavId),
    [navItems, activeNavId]
  );

  const setActiveByIndex = (nextIndex) => {
    if (nextIndex < 0 || nextIndex >= navItems.length) return;
    setActiveNavId(navItems[nextIndex].id);
  };

  const persistRecent = (items) => {
    setRecentStickers(items);
    localStorage.setItem(RECENT_KEY, JSON.stringify(items));
  };

  const onStickerClick = (sticker) => {
    const nextRecent = [sticker, ...recentStickers.filter((s) => s.id !== sticker.id)].slice(0, RECENT_MAX);
    persistRecent(nextRecent);
    hidePopup();
  };

  const onRecentNavWheel = (event) => {
    if (event.deltaY > 0) {
      setActiveByIndex(activeNavIndex + 1);
    } else if (event.deltaY < 0) {
      setActiveByIndex(activeNavIndex - 1);
    }
  };

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
          <X />
        </button>
      </div>

      <div className="popup-grid">
        {loading ? (
          <div className="empty-state">Loading stickers...</div>
        ) : stickers.length === 0 ? (
          <div className="empty-state">No stickers</div>
        ) : (
          stickers.map((sticker) => (
            <button
              key={sticker.id}
              className="sticker-cell"
              type="button"
              onClick={() => onStickerClick(sticker)}
              title={sticker.name}
            >
              <img src={sticker.dataUrl} alt={sticker.name} />
            </button>
          ))
        )}
      </div>

      <div className="popup-nav">
        <button
          className="nav-btn"
          type="button"
          onClick={() => setActiveByIndex(activeNavIndex - 1)}
          disabled={activeNavIndex <= 0}
          title="Back one pack"
        >
          <ChevronLeft />
        </button>

        <div className="nav-strip">
          {navItems.map((item) => (
            <button
              key={item.id}
              type="button"
              className={`nav-pack ${item.id === activeNavId ? "active" : ""}`}
              onClick={() => setActiveNavId(item.id)}
              onWheel={item.id === "recent" ? onRecentNavWheel : undefined}
              title={item.title}
            >
              {item.kind === "recent" ? (
                <span className="nav-recent-icon" aria-hidden="true">
                  <Clock3 />
                </span>
              ) : (
                <img src={item.thumbDataUrl} alt={item.title} />
              )}
            </button>
          ))}
        </div>

        <button
          className="nav-btn"
          type="button"
          onClick={() => setActiveByIndex(activeNavIndex + 1)}
          disabled={activeNavIndex < 0 || activeNavIndex >= navItems.length - 1}
          title="Next one pack"
        >
          <ChevronRight />
        </button>

        <button className="nav-btn add-btn" type="button" title="Add sticker pack">
          <Plus />
        </button>
      </div>
    </div>
  );
}

export default App;
