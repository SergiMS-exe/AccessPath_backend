import { Response, NextFunction } from 'express';
import PlaceModel from '../models/place.model';
import { AuthenticatedRequest } from '../types/express.types';
import { PlaceCreateInput, PlaceUpdateInput, PlaceSearchFilters, AddFeatureInput } from '../types/place.types';

class PlaceController {
  static async create(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const placeData: PlaceCreateInput = {
        ...req.body,
        created_by: req.user.id
      };

      const place = await PlaceModel.create(placeData);

      res.status(201).json({
        message: 'Place created successfully',
        place
      });
    } catch (error) {
      next(error);
    }
  }

  static async getAll(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const filters: PlaceSearchFilters = {
        city: req.query.city as string | undefined,
        category: req.query.category as string | undefined,
        latitude: req.query.lat ? parseFloat(req.query.lat as string) : undefined,
        longitude: req.query.lng ? parseFloat(req.query.lng as string) : undefined,
        radius: req.query.radius ? parseFloat(req.query.radius as string) : undefined,
        limit: req.query.limit ? parseInt(req.query.limit as string) : 50,
        offset: req.query.offset ? parseInt(req.query.offset as string) : 0
      };

      const places = await PlaceModel.findAll(filters);

      res.json({
        count: places.length,
        places
      });
    } catch (error) {
      next(error);
    }
  }

  static async getById(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const place = await PlaceModel.findById(parseInt(req.params.id));

      if (!place) {
        res.status(404).json({ error: 'Place not found' });
        return;
      }

      const features = await PlaceModel.getFeatures(parseInt(req.params.id));

      res.json({
        place: {
          ...place,
          accessibility_features: features
        }
      });
    } catch (error) {
      next(error);
    }
  }

  static async update(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const place = await PlaceModel.findById(parseInt(req.params.id));

      if (!place) {
        res.status(404).json({ error: 'Place not found' });
        return;
      }

      const updateData: PlaceUpdateInput = req.body;
      const updatedPlace = await PlaceModel.update(parseInt(req.params.id), updateData);

      res.json({
        message: 'Place updated successfully',
        place: updatedPlace
      });
    } catch (error) {
      next(error);
    }
  }

  static async delete(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const place = await PlaceModel.findById(parseInt(req.params.id));

      if (!place) {
        res.status(404).json({ error: 'Place not found' });
        return;
      }

      await PlaceModel.delete(parseInt(req.params.id));

      res.json({ message: 'Place deleted successfully' });
    } catch (error) {
      next(error);
    }
  }

  static async addFeature(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const { feature_id, is_available, notes } = req.body as AddFeatureInput;

      const place = await PlaceModel.findById(parseInt(req.params.id));
      if (!place) {
        res.status(404).json({ error: 'Place not found' });
        return;
      }

      const feature = await PlaceModel.addFeature(
        parseInt(req.params.id),
        feature_id,
        is_available ?? true,
        notes ?? null
      );

      res.status(201).json({
        message: 'Feature added to place',
        feature
      });
    } catch (error) {
      next(error);
    }
  }

  static async removeFeature(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const result = await PlaceModel.removeFeature(
        parseInt(req.params.id),
        parseInt(req.params.featureId)
      );

      if (!result) {
        res.status(404).json({ error: 'Feature not found for this place' });
        return;
      }

      res.json({ message: 'Feature removed from place' });
    } catch (error) {
      next(error);
    }
  }
}

export default PlaceController;
