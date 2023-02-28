package database

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/faturohim/ecommerce/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrCantFindProduct    = errors.New("can't find the product")
	ErrcantDecodeProducts = errors.New("cant find the product")
	ErrCantIdIsNotValid   = errors.New("this user is not invalid")
	ErrCantUpdateUser     = errors.New("cannot add this product in the cart")
	ErrCantRemoveItemCart = errors.New("cannot remove this item from the cart")
	ErrCantGetItem        = errors.New("was enable to get the item from the cart")
	ErrCantBuyCartItem    = errors.New("cannot update the purchase")
)

func AddProductToCard(ctx context.Context, prodCollection, userCollection *mongo.Collection, productID primitive.ObjectID, UserID string) error {
	searchfromdb, err := prodCollection.Find(ctx, bson.M{"_id": productID})
	if err != nil {
		log.Println(err)
		return ErrCantFindProduct
	}
	var productCart []models.ProductUser
	err = searchfromdb.All(ctx, &productCart)
	if err != nil {
		log.Println(err)
		return ErrcantDecodeProducts
	}

	id, err := primitive.ObjectIDFromHex(UserID)
	if err != nil {
		log.Println(err)
		return ErrCantIdIsNotValid
	}

	filter := bson.D{primitive.E{Key: "_id", Value: id}}
	update := bson.D{{Key: "$push", Value: bson.D{primitive.E{Key: "usercart", Value: bson.D{{Key: "$each", Value: productCart}}}}}}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return ErrCantUpdateUser
	}
	return nil
}

func RemoveCardItem(ctx context.Context, prodCollection, userCollection *mongo.Collection, productID primitive.ObjectID, UserID string) error {
	id, err := primitive.ObjectIDFromHex(UserID)
	if err != nil {
		log.Println(err)
		return ErrCantIdIsNotValid
	}
	filter := bson.D{primitive.E{Key: "_id", Value: id}}
	update := bson.M{"$pull": bson.M{"usercart": bson.M{"_id": productID}}}
	_, err = userCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return ErrCantRemoveItemCart
	}
	return nil
}

func BuyItemFromCard(ctx context.Context, userCollection *mongo.Collection, UserID string) error {
	// fetch the cart of the user
	// find the cart total
	// create an order with the item
	// added order to the user collection
	// added items in the cart to order list
	// empty up the cart

	id, err := primitive.ObjectIDFromHex(UserID)
	if err != nil {
		log.Println(err)
		return ErrCantIdIsNotValid
	}
	var getcartitems models.User
	var ordercart models.Order

	ordercart.Order_ID = primitive.NewObjectID()
	ordercart.Ordered_At = time.Now()
	ordercart.Order_Cart = make([]models.ProductUser, 0)
	ordercart.Payment_Method.COD = true

	unwind := bson.D{{Key: "$unwind", Value: bson.D{primitive.E{Key: "path", Value: "$usercart"}}}}
	grouping := bson.D{{Key: "$group", Value: bson.D{primitive.E{Key: "_id", Value: "_id"}, {Key: "total", Value: bson.D{primitive.E{Key: "$sum", Value: "$usercart.price"}}}}}}
	currentresult, err := userCollection.Aggregate(ctx, mongo.Pipeline{unwind, grouping})
	ctx.Done()
	if err != nil {
		panic(err)
	}

	var getusercart []bson.M
	if err = currentresult.All(ctx, &getusercart); err != nil {
		panic(err)
	}
	var total_price int32

	for _, user_item := range getusercart {
		price := user_item["total"]
		total_price = price.(int32)
	}
	ordercart.Price = int(total_price)

	filter := bson.D{primitive.E{Key: "_id", Value: id}}
	update := bson.D{{Key: "$push", Value: bson.D{primitive.E{Key: "orders", Value: ordercart}}}}
	_, err = userCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Println(err)
	}
	err = userCollection.FindOne(ctx, bson.D{primitive.E{Key: "_id", Value: id}}).Decode(&getcartitems)
	if err != nil {
		log.Println(err)
	}

	filter1 := bson.D{primitive.E{Key: "_id", Value: id}}
	update2 := bson.M{"$push": bson.M{"orders.$[].order_list": bson.M{"$each": getcartitems.UserCart}}}
	_, err = userCollection.UpdateOne(ctx, filter1, update2)
	if err != nil {
		log.Println(err)
	}

	usercart_empty := make([]models.ProductUser, 0)
	filter3 := bson.D{primitive.E{Key: "_id", Value: id}}
	update3 := bson.D{{Key: "$set", Value: bson.D{primitive.E{Key: "usercart", Value: usercart_empty}}}}
	_, err = userCollection.UpdateOne(ctx, filter3, update3)
	if err != nil {
		return ErrCantBuyCartItem
	}
	return nil
}

func InstantBuyer(ctx context.Context, prodCollection, userCollection *mongo.Collection, productID primitive.ObjectID, UserID string) error {
	id, err := primitive.ObjectIDFromHex(UserID)

	if err != nil {
		log.Println(err)
		return ErrCantIdIsNotValid
	}
	var product_details models.ProductUser
	var orders_detail models.Order

	orders_detail.Order_ID = primitive.NewObjectID()
	orders_detail.Ordered_At = time.Now()
	orders_detail.Order_Cart = make([]models.ProductUser, 0)
	orders_detail.Payment_Method.COD = true
	err = prodCollection.FindOne(ctx, bson.D{primitive.E{Key: "_id", Value: productID}}).Decode(&product_details)
	if err != nil {
		log.Println(err)
	}
	orders_detail.Price = product_details.Price

	filter := bson.D{primitive.E{Key: "_id", Value: id}}
	update := bson.D{{Key: "$push", Value: bson.D{primitive.E{Key: "orders", Value: orders_detail}}}}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println(err)
	}

	filter1 := bson.D{primitive.E{Key: "_id", Value: id}}
	update1 := bson.M{"$push": bson.M{"orders.$[].order_list": product_details}}

	_, err = userCollection.UpdateOne(ctx, filter1, update1)
	if err != nil {
		log.Println(err)
	}
	return nil
}
