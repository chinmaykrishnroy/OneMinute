import type { SVGProps } from "react";
export type IconName = "discover" | "messages" | "moments" | "activity" | "posts" | "profile" | "settings" | "connections" | "shield" | "logout" | "arrow" | "close" | "phone" | "video" | "back";
const paths: Record<IconName, string> = {
 moments: "m12 3 2.7 6.3L21 12l-6.3 2.7L12 21l-2.7-6.3L3 12l6.3-2.7L12 3Z",
 activity: "M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4",
 phone: "M7 3H4a1 1 0 0 0-1 1c0 9.4 7.6 17 17 17a1 1 0 0 0 1-1v-3l-5-2-2 2a14 14 0 0 1-7-7l2-2-2-5Z",
 video: "M4 5h10a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2ZM16 10l6-4v12l-6-4",
 back: "M19 12H5m6-6-6 6 6 6",
 discover: "m16.5 7.5-3 6-6 3 3-6 6-3ZM12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20",
 messages: "M21 11.5a8.4 8.4 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.4 8.4 0 0 1-3.8-.9L3 21l1.9-5.7a8.4 8.4 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.4 8.4 0 0 1 3.8-.9H13a8.5 8.5 0 0 1 8 8v.5ZM8 11h8M8 15h4",
 posts: "M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2ZM12 8v8M8 12h8",
 profile: "M20 21v-2a7 7 0 0 0-14 0v2M12 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8",
 settings: "M4 6h16M4 12h16M4 18h16M8 3v6M16 9v6M9 15v6",
 connections: "M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75",
 shield: "M12 22s8-4 8-11V5l-8-3-8 3v6c0 7 8 11 8 11ZM9 9l6 6M15 9l-6 6",
 logout: "M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9",
 arrow: "M5 12h14M13 6l6 6-6 6",
 close: "M6 6l12 12M18 6 6 18",
};
export function Icon({ name, ...props }: SVGProps<SVGSVGElement> & { name: IconName }) {
 return <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" {...props}><path d={paths[name]} /></svg>;
}
