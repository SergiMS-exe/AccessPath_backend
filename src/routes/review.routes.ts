import { Router } from 'express';
import { body } from 'express-validator';
import ReviewController from '../controllers/review.controller';
import authMiddleware from '../middleware/auth';
import validate from '../middleware/validation';

const router = Router();

router.get('/place/:placeId', ReviewController.getByPlaceId);

router.post(
  '/',
  authMiddleware,
  [
    body('place_id').isInt().withMessage('Valid place_id is required'),
    body('rating').isInt({ min: 1, max: 5 }).withMessage('Rating must be between 1 and 5'),
    body('comment').optional().isString(),
    validate
  ],
  ReviewController.create
);

router.put(
  '/:id',
  authMiddleware,
  [
    body('rating').isInt({ min: 1, max: 5 }).withMessage('Rating must be between 1 and 5'),
    body('comment').optional().isString(),
    validate
  ],
  ReviewController.update
);

router.delete('/:id', authMiddleware, ReviewController.delete);

export default router;
