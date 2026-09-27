export type Rotation = 0 | 90 | 180 | 270;

export interface RotatePage {
  page: number;
  preview: string;
  rotation: Rotation;
}
