export type Profile = { id: string; displayName: string; avatarUrl: string; bio: string; countryCode: string; interests: string[]; languages: string[]; discoveryIntent: string };
export const intents = [["surprise_me", "Surprise me"], ["new_friends", "New friends"], ["dating", "Dating"], ["gaming", "Gaming"], ["language_exchange", "Language exchange"], ["tech_ideas", "Tech / ideas"], ["professional_networking", "Professional networking"]];
export const languages = [["en", "English"], ["hi", "Hindi"], ["bn", "Bengali"], ["es", "Spanish"], ["fr", "French"], ["de", "German"], ["ja", "Japanese"]];
export const interests = ["ai", "art", "books", "films", "fitness", "gaming", "music", "nature", "photography", "science", "technology", "travel"];
export function profileReady(profile: Profile) { return Boolean(profile.displayName.trim() && profile.discoveryIntent && profile.languages.length); }
export function interestLabel(value: string) { return value === "ai" ? "AI" : value.charAt(0).toUpperCase() + value.slice(1); }
