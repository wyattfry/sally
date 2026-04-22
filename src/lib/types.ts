export type CapturedPage = {
  title: string;
  url: string;
  visibleText: string;
  mainImageUrl?: string;
  structuredData: unknown[];
  pdfLinks: string[];
};

export type ScheduleItem = {
  id: string;
  capturedAt: string;
  zone: string;
  title: string;
  manufacturer: string;
  modelNumber: string;
  category: string;
  description: string;
  finish: string;
  finishModelNumber?: string;
  requiredAddOns: string[];
  optionalCompanions: string[];
  sourceUrl: string;
  sourceTitle: string;
  sourceImageUrl?: string;
  sourcePdfLinks: string[];
};

