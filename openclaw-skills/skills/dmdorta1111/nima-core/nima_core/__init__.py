"""NIMA Core — Dynamic Affect System for AI agents."""

__version__ = "2.0.8"

from .cognition.dynamic_affect import DynamicAffectSystem, AffectVector, get_affect_system
from .cognition.personality_profiles import PersonalityManager, get_profile, list_profiles
from .cognition.emotion_detection import map_emotions_to_affects, detect_affect_from_text
from .cognition.response_modulator_v2 import GenericResponseModulator, ResponseGuidance, modulate_response
from .cognition.archetypes import (
    ARCHETYPES,
    get_archetype,
    list_archetypes,
    baseline_from_archetype,
    baseline_from_description
)

__all__ = [
    "DynamicAffectSystem",
    "AffectVector",
    "get_affect_system",
    "PersonalityManager",
    "get_profile",
    "list_profiles",
    "map_emotions_to_affects",
    "detect_affect_from_text",
    "GenericResponseModulator",
    "ResponseGuidance",
    "modulate_response",
    "ARCHETYPES",
    "get_archetype",
    "list_archetypes",
    "baseline_from_archetype",
    "baseline_from_description",
    "__version__",
]