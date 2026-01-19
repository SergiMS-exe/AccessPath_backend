import { Response, NextFunction } from 'express';
import ReviewModel from '../models/review.model';
import PlaceModel from '../models/place.model';
import { AuthenticatedRequest } from '../types/express.types';
import { ReviewCreateInput, ReviewUpdateInput } from '../types/review.types';

class ReviewController {
  static async create(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const { place_id, rating, comment } = req.body;

      const place = await PlaceModel.findById(place_id);
      if (!place) {
        res.status(404).json({ error: 'Place not found' });
        return;
      }

      const reviewData: ReviewCreateInput = {
        place_id,
        user_id: req.user.id,
        rating,
        comment
      };

      const review = await ReviewModel.create(reviewData);

      res.status(201).json({
        message: 'Review created successfully',
        review
      });
    } catch (error) {
      next(error);
    }
  }

  static async getByPlaceId(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const reviews = await ReviewModel.findByPlaceId(parseInt(req.params.placeId));

      res.json({
        count: reviews.length,
        reviews
      });
    } catch (error) {
      next(error);
    }
  }

  static async update(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const review = await ReviewModel.findById(parseInt(req.params.id));

      if (!review) {
        res.status(404).json({ error: 'Review not found' });
        return;
      }

      if (review.user_id !== req.user.id) {
        res.status(403).json({ error: 'Not authorized to update this review' });
        return;
      }

      const updateData: ReviewUpdateInput = {
        rating: req.body.rating,
        comment: req.body.comment
      };

      const updatedReview = await ReviewModel.update(parseInt(req.params.id), updateData);

      res.json({
        message: 'Review updated successfully',
        review: updatedReview
      });
    } catch (error) {
      next(error);
    }
  }

  static async delete(req: AuthenticatedRequest, res: Response, next: NextFunction): Promise<void> {
    try {
      const review = await ReviewModel.findById(parseInt(req.params.id));

      if (!review) {
        res.status(404).json({ error: 'Review not found' });
        return;
      }

      if (review.user_id !== req.user.id) {
        res.status(403).json({ error: 'Not authorized to delete this review' });
        return;
      }

      await ReviewModel.delete(parseInt(req.params.id));

      res.json({ message: 'Review deleted successfully' });
    } catch (error) {
      next(error);
    }
  }
}

export default ReviewController;
