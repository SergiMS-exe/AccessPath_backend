import { Request, Response, NextFunction } from 'express';
import FeatureModel from '../models/feature.model';

class FeatureController {
  static async getAll(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const features = await FeatureModel.findAll();

      res.json({
        count: features.length,
        features
      });
    } catch (error) {
      next(error);
    }
  }

  static async getById(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const feature = await FeatureModel.findById(parseInt(req.params.id));

      if (!feature) {
        res.status(404).json({ error: 'Feature not found' });
        return;
      }

      res.json({ feature });
    } catch (error) {
      next(error);
    }
  }

  static async getByCategory(req: Request, res: Response, next: NextFunction): Promise<void> {
    try {
      const features = await FeatureModel.findByCategory(req.params.category);

      res.json({
        count: features.length,
        features
      });
    } catch (error) {
      next(error);
    }
  }
}

export default FeatureController;
