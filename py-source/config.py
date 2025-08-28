import json
import os
from typing import Any, Dict, Optional


def default_config_path() -> Optional[str]:
    # Check env var first
    env = os.environ.get("LC_KNOWLEDGE_CONFIG")
    if env and os.path.exists(env):
        return env
    # Then local config.json
    local = os.path.join(os.getcwd(), "config.json")
    if os.path.exists(local):
        return local
    # Optionally look in XDG
    xdg_home = os.environ.get("XDG_CONFIG_HOME") or os.path.join(
        os.path.expanduser("~"), ".config"
    )
    xdg_path = os.path.join(xdg_home, "lc-knowledge", "config.json")
    if os.path.exists(xdg_path):
        return xdg_path
    return None


def load_config(path: Optional[str]) -> Dict[str, Any]:
    if not path:
        return {}
    try:
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)
    except FileNotFoundError:
        return {}
    except Exception:
        # If malformed, ignore to not block CLI
        return {}


def cfg_get(cfg: Dict[str, Any], dotted: str, default: Any = None) -> Any:
    cur = cfg
    for part in dotted.split("."):
        if not isinstance(cur, dict) or part not in cur:
            return default
        cur = cur[part]
    return cur

