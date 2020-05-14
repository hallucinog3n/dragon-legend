package npc

import (
	"github.com/syntaxgame/dragon-legend/database"
	"github.com/syntaxgame/dragon-legend/utils"
)

type BuyItemHandler struct {
}

type SellItemHandler struct {
}

var ()

func (h *BuyItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	itemID := utils.BytesToInt(data[6:10], true)
	quantity := utils.BytesToInt(data[10:12], true)
	slotID := int16(utils.BytesToInt(data[16:18], true))

	npcID := int(utils.BytesToInt(data[18:22], true))
	shopID, ok := shops[npcID]
	if !ok {
		shopID = 25
	}

	shop, ok := database.Shops[shopID]
	if !ok {
		return nil, nil
	}

	canPurchase := shop.IsPurchasable(int(itemID))
	if !canPurchase {
		return nil, nil
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	info := database.Items[itemID]
	cost := uint64(info.BuyPrice) * uint64(quantity)
	if slots[slotID].ItemID == 0 && cost <= c.Gold && quantity > 0 { // slot is empty, player can afford and quantity is positive
		c.LootGold(-cost)
		item := &database.InventorySlot{ItemID: itemID, Quantity: uint(quantity)}

		if info.GetType() == database.PET_TYPE {
			petInfo := database.Pets[item.ItemID]
			expInfo := database.PetExps[petInfo.Level-1]

			item.Pet = &database.PetSlot{
				Fullness: 100, Loyalty: 100,
				Exp:   uint64(expInfo.ReqExpEvo1),
				HP:    petInfo.BaseHP,
				Level: byte(petInfo.Level),
				Name:  petInfo.Name,
				CHI:   petInfo.BaseChi}
		}

		resp, _, err := c.AddItem(item, slotID, false)
		if err != nil {
			return nil, err
		} else if resp == nil {
			return nil, nil
		}

		return *resp, nil
	}

	return nil, nil
}

func (h *SellItemHandler) Handle(s *database.Socket, data []byte) ([]byte, error) {

	c := s.Character
	if c == nil {
		return nil, nil
	}

	slots, err := c.InventorySlots()
	if err != nil {
		return nil, err
	}

	c.Looting.Lock()
	defer c.Looting.Unlock()

	itemID := utils.BytesToInt(data[6:10], true)
	quantity := int(utils.BytesToInt(data[10:12], true))
	slotID := int16(utils.BytesToInt(data[12:14], true))

	item := database.Items[itemID]
	slot := slots[slotID]

	if !item.Tradable {
		return nil, nil
	}

	multiplier := 0
	if slot.ItemID == itemID && quantity > 0 && uint(quantity) <= slot.Quantity {
		upgs := slot.GetUpgrades()
		for i := uint8(0); i < slot.Plus; i++ {
			upg := upgs[i]
			if code, ok := database.HaxCodes[int(upg)]; ok {
				multiplier += code.SaleMultiplier
			}
		}

		multiplier /= 1000
		if multiplier == 0 {
			multiplier = 1
		}

		unitPrice := uint64(item.SellPrice) * uint64(multiplier)
		if slot.Plus > 0 {
			unitPrice *= uint64(slot.Plus)
		}

		return c.SellItem(int(itemID), int(slotID), int(quantity), unitPrice)
	}

	return nil, nil
}
